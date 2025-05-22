package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/go-co-op/gocron/v2"
	"github.com/jxs13/league-discord-bot/internal/format"
	"github.com/jxs13/league-discord-bot/internal/parse"
	"github.com/jxs13/league-discord-bot/internal/timeutils"
	"github.com/jxs13/league-discord-bot/sqlc"
)

type Bot struct {
	ctx         context.Context
	cancelCause context.CancelCauseFunc
	state       *state.State
	db          *sql.DB
	userID      discord.UserID
	wg          *sync.WaitGroup

	defaultNotificationOffsets []time.Duration
	defaultChannelAccessOffset time.Duration
	defaulRequirementsOffset   time.Duration
	defaultChannelDeleteOffset time.Duration

	backupDir      string
	backupFile     string
	backupInterval time.Duration

	scheduler gocron.Scheduler

	jobMu                       sync.Mutex
	announcementJob             gocron.Job
	channelAccessJob            gocron.Job
	channelDeleteJob            gocron.Job
	notificationsJob            gocron.Job
	participationRequirementJob gocron.Job
}

type JobDefinition struct {
	Scale time.Duration
	Def   gocron.JobDefinition
}

var (
	scales = []JobDefinition{
		{
			Scale: 720 * time.Hour,
			Def:   gocron.MonthlyJob(1, gocron.NewDaysOfTheMonth(1), gocron.NewAtTimes(gocron.NewAtTime(2, 0, 0))),
		},
		{
			Scale: 168 * time.Hour,
			Def:   gocron.WeeklyJob(1, gocron.NewWeekdays(time.Monday), gocron.NewAtTimes(gocron.NewAtTime(2, 0, 0))),
		},
		{
			Scale: 24 * time.Hour,
			Def:   gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(2, 0, 0))),
		},
		{
			Scale: 12 * time.Hour,
			Def: gocron.DailyJob(1, gocron.NewAtTimes(
				gocron.NewAtTime(2, 0, 0),
				gocron.NewAtTime(14, 0, 0),
			)),
		},
		{
			Scale: 6 * time.Hour,
			Def: gocron.DailyJob(1, gocron.NewAtTimes(
				gocron.NewAtTime(2, 0, 0),
				gocron.NewAtTime(8, 0, 0),
				gocron.NewAtTime(14, 0, 0),
				gocron.NewAtTime(20, 0, 0),
			)),
		},
	}
)

func SelectJobDefinition(interval time.Duration, factor ...int) gocron.JobDefinition {
	n := time.Duration(1)
	if len(factor) > 0 && factor[0] > 1 {
		n = time.Duration(factor[0])
	}

	for _, s := range scales {
		if interval%(s.Scale*n) == 0 {
			return s.Def
		}
	}
	return gocron.DurationJob(interval * n)
}

// New requires a discord bot token and returns a Bot instance.
// A bot token starts with Nj... and can be obtained from the discord developer portal.
func New(
	ctx context.Context,
	token string,
	db *sql.DB,
	defaultNotificationOffsets []time.Duration,
	defaultChannelAccessOffset time.Duration,
	defaulRequirementsOffset time.Duration,
	defaultChannelDeleteOffset time.Duration,
	backupDir string,
	backupFile string,
	backupInterval time.Duration,
) (*Bot, error) {

	scheduler, err := gocron.NewScheduler(gocron.WithLimitConcurrentJobs(1, gocron.LimitModeWait))
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}

	ctx, cancelCause := context.WithCancelCause(ctx)

	s := state.New("Bot " + token)
	bot := &Bot{
		ctx:                        ctx,
		cancelCause:                cancelCause,
		state:                      s,
		db:                         db,
		wg:                         &sync.WaitGroup{},
		defaultNotificationOffsets: defaultNotificationOffsets,
		defaultChannelAccessOffset: defaultChannelAccessOffset,
		defaulRequirementsOffset:   defaulRequirementsOffset,
		defaultChannelDeleteOffset: defaultChannelDeleteOffset,
		scheduler:                  scheduler,
		backupDir:                  backupDir,
		backupFile:                 backupFile,
		backupInterval:             backupInterval,
	}

	s.AddIntents(
		gateway.IntentGuilds | gateway.IntentGuildMessages | gateway.IntentGuildMessageReactions | gateway.IntentGuildScheduledEvents,
	)

	var startupOnce sync.Once
	s.AddHandler(func(*gateway.ReadyEvent) {
		// it's possible that the bot occasionally looses the gateway connection
		// calling this heavy weight function on every reconnect is not ideal
		startupOnce.Do(func() {

			me, err := s.Me()
			if err != nil {
				bot.cancelCause(fmt.Errorf("failed to get bot user: %w", err))
				return
			}
			bot.userID = me.ID

			// print statistics on startup
			bot.printDailyStatistics()

			_, err = bot.scheduler.NewJob(
				gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 0, 0))),
				gocron.NewTask(bot.printDailyStatistics),
			)
			if err != nil {
				bot.cancelCause(fmt.Errorf("failed to create daily statistics job: %w", err))
				return
			}

			if bot.backupInterval > 0 {
				_, err = bot.scheduler.NewJob(
					SelectJobDefinition(bot.backupInterval),
					gocron.NewTask(bot.createBackup),
				)
				if err != nil {
					bot.cancelCause(fmt.Errorf("failed to create backup job: %w", err))
					return
				}

				// cleanup every month
				_, err = bot.scheduler.NewJob(
					gocron.MonthlyJob(1,
						gocron.NewDaysOfTheMonth(1),
						gocron.NewAtTimes(
							gocron.NewAtTime(2, 0, 0),
						),
					),
					gocron.NewTask(bot.compressBackups),
				)
				if err != nil {
					bot.cancelCause(fmt.Errorf("failed to create backup compression job: %w", err))
					return
				}
			}

			err = bot.TxQueries(ctx, bot.refreshJobSchedules)
			if err != nil {
				bot.cancelCause(fmt.Errorf("failed to initially refresh job schedules: %w", err))
				return
			}
		})
	})

	// requires guild message intents

	s.AddHandler(bot.handleAddGuild)
	s.AddHandler(bot.handleRemoveGuild)

	s.AddHandler(bot.handleChannelDelete)
	s.AddHandler(bot.handleAddParticipationReaction)
	s.AddHandler(bot.handleRemoveParticipationReaction)

	s.AddHandler(bot.handleScheduledEventDelete)
	s.AddHandler(bot.handleScheduledEventUpdate)

	s.AddHandler(bot.handleAutocompletionLocationInteraction)

	r := cmdroute.NewRouter()
	// Automatically defer handles if they're slow.
	r.Use(cmdroute.Deferrable(s, cmdroute.DeferOpts{
		Flags: discord.EphemeralMessage,
	}))

	// admin commands
	r.AddFunc("configure", bot.commandGuildConfigure)
	r.AddFunc("configuration", bot.commandGuildConfiguration)

	// admin + user commands
	r.AddFunc("schedule-match", bot.commandScheduleMatch)

	r.AddFunc("notification-list", bot.commandNotificationsList)
	r.AddFunc("notification-delete", bot.commandNotificationsDelete)
	r.AddFunc("notification-add", bot.commandNotificationsAdd)

	r.AddFunc("announcements-enable", bot.commandAnnouncementsEnable)
	r.AddFunc("announcements-disable", bot.commandAnnouncementsDisable)
	r.AddFunc("announcements-configuration", bot.commandAnnouncementConfiguration)

	s.AddInteractionHandler(r)

	err = bot.overrideCommands()
	if err != nil {
		return nil, fmt.Errorf("failed to override commands: %w", err)
	}

	return bot, nil
}

func (b *Bot) Connect(ctx context.Context) error {
	b.scheduler.Start()
	return b.state.Connect(ctx)
}

func (b *Bot) Close() error {
	b.cancelCause(errors.New("bot closed"))
	defer b.wg.Wait()

	return errors.Join(
		b.state.Close(),
		b.scheduler.Shutdown(),
	)
}

func (b *Bot) refreshAccessJob(ctx context.Context, q *sqlc.Queries) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to refresh access job: %w", err)
		}
	}()
	accessible, err := q.NextAccessibleChannel(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to get next accessible channel: %w", err)
	}

	b.jobMu.Lock()
	defer b.jobMu.Unlock()

	b.channelAccessJob, err = b.rescheduleJob(
		b.channelAccessJob,
		accessible.ChannelAccessible,
		b.asyncGrantChannelAccess,
	)
	if err != nil {
		return fmt.Errorf("failed to reschedule channel access job: %w", err)
	}

	return nil
}

func (b *Bot) refreshNotificationJob(ctx context.Context, q *sqlc.Queries) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to refresh notification job: %w", err)
		}
	}()
	notification, err := q.NextNotification(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to get next notification: %w", err)
	}

	b.jobMu.Lock()
	defer b.jobMu.Unlock()

	b.notificationsJob, err = b.rescheduleJob(
		b.notificationsJob,
		notification.NotifyAt,
		b.asyncNotifications,
	)
	if err != nil {
		return fmt.Errorf("failed to reschedule notifications job: %w", err)
	}

	return nil
}

func (b *Bot) refreshParticipationRequirementJob(ctx context.Context, q *sqlc.Queries) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to refresh participation requirement job: %w", err)
		}
	}()
	participationRequirement, err := q.NextParticipationRequirement(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to get next participation requirement: %w", err)
	}

	b.jobMu.Lock()
	defer b.jobMu.Unlock()

	b.participationRequirementJob, err = b.rescheduleJob(
		b.participationRequirementJob,
		participationRequirement.DeadlineAt,
		b.asyncCheckParticipationDeadline,
	)
	if err != nil {
		return fmt.Errorf("failed to reschedule participation requirement job: %w", err)
	}

	return nil
}

func (b *Bot) refreshChannelDeleteJob(ctx context.Context, q *sqlc.Queries) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to refresh channel delete job: %w", err)
		}
	}()
	deletable, err := q.NextDeletableChannel(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to get next deletable channel: %w", err)
	}

	b.jobMu.Lock()
	defer b.jobMu.Unlock()

	b.channelDeleteJob, err = b.rescheduleJob(
		b.channelDeleteJob,
		deletable.ChannelDeleteAt,
		b.asyncDeleteExpiredChannels,
	)
	if err != nil {
		return fmt.Errorf("failed to reschedule channel delete job: %w", err)
	}

	return nil
}

func (b *Bot) refreshAnnouncementJob(ctx context.Context, q *sqlc.Queries) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to refresh announcement job: %w", err)
		}
	}()
	announcement, err := q.NextAnnouncement(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to get next announcement: %w", err)
	}

	b.jobMu.Lock()
	defer b.jobMu.Unlock()

	b.announcementJob, err = b.rescheduleJob(
		b.announcementJob,
		announcement.LastAnnouncedAt+announcement.Interval,
		b.asyncAnnouncements,
	)
	if err != nil {
		return fmt.Errorf("failed to reschedule announcement job: %w", err)
	}

	return nil
}

func (b *Bot) refreshJobSchedules(ctx context.Context, q *sqlc.Queries) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to refresh job schedules: %w", err)
		}
	}()
	accessible, err := q.NextAccessibleChannel(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to get next accessible channel: %w", err)
	}

	deletable, err := q.NextDeletableChannel(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to get next deletable channel: %w", err)
	}

	notification, err := q.NextNotification(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to get next notification: %w", err)
	}

	participationRequirement, err := q.NextParticipationRequirement(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to get next participation requirement: %w", err)

	}

	announcement, err := q.NextAnnouncement(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to get next announcement: %w", err)
	}

	b.jobMu.Lock()
	defer b.jobMu.Unlock()

	b.channelAccessJob, err = b.rescheduleJob(
		b.channelAccessJob,
		accessible.ChannelAccessible,
		b.asyncGrantChannelAccess,
	)
	if err != nil {
		return fmt.Errorf("failed to reschedule channel access job: %w", err)
	}

	b.channelDeleteJob, err = b.rescheduleJob(
		b.channelDeleteJob,
		deletable.ChannelDeleteAt,
		b.asyncDeleteExpiredChannels,
	)
	if err != nil {
		return fmt.Errorf("failed to reschedule channel delete job: %w", err)
	}

	b.notificationsJob, err = b.rescheduleJob(
		b.notificationsJob,
		notification.NotifyAt,
		b.asyncNotifications,
	)
	if err != nil {
		return fmt.Errorf("failed to reschedule notifications job: %w", err)
	}

	b.participationRequirementJob, err = b.rescheduleJob(
		b.participationRequirementJob,
		participationRequirement.DeadlineAt,
		b.asyncCheckParticipationDeadline,
	)
	if err != nil {
		return fmt.Errorf("failed to reschedule participation requirement job: %w", err)
	}

	b.announcementJob, err = b.rescheduleJob(
		b.announcementJob,
		announcement.LastAnnouncedAt+announcement.Interval,
		b.asyncAnnouncements,
	)
	if err != nil {
		return fmt.Errorf("failed to reschedule announcement job: %w", err)
	}

	return nil
}

func (b *Bot) rescheduleJob(job gocron.Job, epoch int64, function any, parameters ...any) (_ gocron.Job, err error) {
	if max(epoch, 0) == 0 {
		if job != nil {
			log.Println("disabling job", funcName(job.Name()))
		}
		return nil, nil
	}

	if job == nil {
		return b.newUnixJob(epoch, function, parameters...)
	}

	// wait 30 seconds more in order to fetch more db entries at once.
	epochAdjusted := timeutils.Ceil(time.Unix(epoch, 0), time.Minute/2).Unix()

	nextRun, err := job.NextRun()
	if err == nil && !nextRun.IsZero() {
		if nextRun.Unix() <= epochAdjusted {
			// no need to reschedule
			//log.Println("not updating job", funcName(job.Name()), "starting at", time.Unix(nextRun.Unix(), 0), "to", time.Unix(epochAdjusted, 0))
			return job, nil
		}
	}

	log.Println("rescheduling job", funcName(job.Name()), "to", time.Unix(epochAdjusted, 0))
	return b.scheduler.Update(
		job.ID(),
		newJobDefinitionUnix(epochAdjusted),
		gocron.NewTask(function, parameters...),
	)
}

func (b *Bot) newUnixJob(epoch int64, function any, parameters ...any) (gocron.Job, error) {
	j, err := b.scheduler.NewJob(
		newJobDefinitionUnix(epoch),
		gocron.NewTask(function, parameters...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create new job: %w", err)
	}

	log.Println("scheduling job", funcName(j.Name()), "at", time.Unix(epoch, 0))
	return j, nil
}

func funcName(f string) string {
	parts := strings.Split(f, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return f
}

func newJobDefinitionUnix(epoch int64) gocron.JobDefinition {
	// when not in the future, schedule immediately.
	nowUnix := time.Now().Unix()
	if epoch <= nowUnix {
		return gocron.OneTimeJob(gocron.OneTimeJobStartImmediately())
	}
	return gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(time.Unix(epoch, 0)))
}

func (b *Bot) isMe(userID discord.UserID) bool {
	return userID == b.userID
}

// Used as database value
func (b *Bot) DefaultReminderIntervals() string {
	return format.ReminderIntervals(b.defaultNotificationOffsets)
}

func errorResponse(err error) *api.InteractionResponseData {
	log.Println(err)
	return &api.InteractionResponseData{
		Content:         option.NewNullableString("**Error:** " + err.Error()),
		Flags:           discord.EphemeralMessage,
		AllowedMentions: &api.AllowedMentions{ /* none */ },
	}
}

func (b *Bot) TxQueries(ctx context.Context, f func(ctx context.Context, q *sqlc.Queries) error) error {
	tx, err := b.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, tx.Rollback())
		}
	}()

	d := sqlc.New(tx)
	defer d.Close()
	err = f(ctx, d)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (b *Bot) Queries(ctx context.Context, f func(ctx context.Context, q *sqlc.Queries) error) (err error) {
	conn, err := b.db.Conn(ctx)
	if err != nil {
		return err
	}
	// return into pool in order to not exhaust the pool
	defer conn.Close()
	q := sqlc.New(conn)
	defer q.Close()
	return f(ctx, q)
}

func (b *Bot) overrideCommands() error {
	var userCommandList = []api.CreateCommandData{
		{
			Name:           "configuration",
			Description:    "Get the current server configuration",
			NoDMPermission: true,
			DefaultMemberPermissions: discord.NewPermissions(
				discord.PermissionAdministrator,
			),
			Options: []discord.CommandOption{},
		},
		{
			Name:           "configure",
			Description:    "Configure the bot for the current guild",
			NoDMPermission: true,
			DefaultMemberPermissions: discord.NewPermissions(
				discord.PermissionAdministrator,
			),
			Options: []discord.CommandOption{
				&discord.BooleanOption{
					OptionName:  "enabled",
					Description: "Enable or disable the bot for this server",
				},
				&discord.StringOption{
					OptionName:  "channel_access_offset",
					Description: "How long before the match the user can access the mach channel",
					MinLength:   option.NewInt(2),
					MaxLength:   option.NewInt(11),
				},
				&discord.BooleanOption{
					OptionName:  "event_creation_enabled",
					Description: "Automatically create scheduled events for matches that have a streamer and stream_url",
				},
				&discord.StringOption{
					OptionName:  "notification_offsets",
					Description: "Intervals at which to remind before a match e.g. 24h,1h,15m,5m,30s or empty for no defaults",
				},
				&discord.StringOption{
					OptionName:  "requirements_offset",
					Description: "Time before the match until which the participation requirements need to be met e.g. 24h, 30m, 0s",
					MinLength:   option.NewInt(2),
					MaxLength:   option.NewInt(11),
				},
				&discord.StringOption{
					OptionName:  "channel_delete_offset",
					Description: "Deadline after the match at which the channel is deleted e.g. 1h, 0s, 50m, 1h50m,30s",
					MinLength:   option.NewInt(2),
					MaxLength:   option.NewInt(11),
				},
			},
		},
		{
			Name:           "schedule-match",
			Description:    "Schedule a new match",
			NoDMPermission: true,
			DefaultMemberPermissions: discord.NewPermissions(
				discord.PermissionViewChannel,
				discord.PermissionSendMessages,
			),

			Options: []discord.CommandOption{
				&discord.StringOption{
					OptionName:  "scheduled_at",
					Description: fmt.Sprintf("Time when the match starts. Must be in this format: %s", parse.LayoutDateTime),
					MinLength:   option.NewInt(len(parse.LayoutDateTime)),
					MaxLength:   option.NewInt(len(parse.LayoutDateTime)),
					Required:    true,
				},
				&discord.StringOption{
					OptionName:   "location",
					Description:  "Timzone location, e.g. Europe/Berlin.",
					MinLength:    option.NewInt(1),
					Required:     true,
					Autocomplete: true,
				},
				&discord.RoleOption{
					OptionName:  "team_1_role",
					Description: "Role of the first team.",
					Required:    true,
				},
				&discord.RoleOption{
					OptionName:  "team_2_role",
					Description: "Role of the second team.",
					Required:    true,
				},
				&discord.UserOption{
					OptionName:  "moderator",
					Description: "Moderator",
					Required:    true,
				},
				&discord.IntegerOption{
					OptionName:  "participants_per_team",
					Description: "Number of required participants per team. (3on3 -> 3)",
					Min:         option.NewInt(0),
					Required:    false,
				},
				&discord.UserOption{
					OptionName:  "streamer",
					Description: "Streamer",
					Required:    false,
				},
				&discord.StringOption{
					OptionName:  "stream_url",
					Description: "url of the streamer or stream",
					Required:    false,
				},
			},
		},
		{
			Name:           "notification-list",
			Description:    "list all notifications for a specific match",
			NoDMPermission: true,
			DefaultMemberPermissions: discord.NewPermissions(
				discord.PermissionViewChannel,
				discord.PermissionSendMessages,
			),

			Options: []discord.CommandOption{
				&discord.ChannelOption{
					OptionName:  "match_channel",
					Description: "Match channel for which to get the notifications",
					Required:    true,
				},
			},
		},
		{
			Name:           "notification-delete",
			Description:    "delete a notification from the notification list",
			NoDMPermission: true,
			DefaultMemberPermissions: discord.NewPermissions(
				discord.PermissionViewChannel,
				discord.PermissionSendMessages,
			),

			Options: []discord.CommandOption{
				&discord.ChannelOption{
					OptionName:  "match_channel",
					Description: "Match channel for which to get the notification",
					Required:    true,
				},
				&discord.IntegerOption{
					OptionName:  "list_number",
					Description: "Notification number in the notification list",
					Required:    true,
					Min:         option.NewInt(1),
					Max:         option.NewInt(50),
				},
			},
		},
		{
			Name:           "notification-add",
			Description:    "add a new generated or custom notification to a match channel.",
			NoDMPermission: true,
			DefaultMemberPermissions: discord.NewPermissions(
				discord.PermissionViewChannel,
				discord.PermissionSendMessages,
			),

			Options: []discord.CommandOption{
				&discord.ChannelOption{
					OptionName:  "match_channel",
					Description: "Match channel for which to get the notification",
					Required:    true,
				},
				&discord.StringOption{
					OptionName:  "notify_at",
					Description: fmt.Sprintf("Time when the notification is triggered. Must be in this format: %s", parse.LayoutDateTime),
					MinLength:   option.NewInt(len(parse.LayoutDateTime)),
					MaxLength:   option.NewInt(len(parse.LayoutDateTime)),
					Required:    true,
				},
				&discord.StringOption{
					OptionName:   "location",
					Description:  "Timzone location, e.g. Europe/Berlin.",
					MinLength:    option.NewInt(1),
					Required:     true,
					Autocomplete: true,
				},
				&discord.StringOption{
					OptionName:  "custom_text",
					Description: "Leave empty for a default generated message",
					Required:    false,
				},
			},
		},
		{
			Name:           "announcements-disable",
			Description:    "Disable periodic (daily, weekly, monthly, etc.) announcements of scheduled matches",
			NoDMPermission: true,
			DefaultMemberPermissions: discord.NewPermissions(
				discord.PermissionAdministrator,
			),
		},
		{
			Name:           "announcements-configuration",
			Description:    "Get the current server announcement configuration",
			NoDMPermission: true,
			DefaultMemberPermissions: discord.NewPermissions(
				discord.PermissionAdministrator,
			),
		},
		{
			Name:           "announcements-enable",
			Description:    "Enable periodic announcements hourly, daily, weekly, monthly, etc. ahead of scheduled matches",
			NoDMPermission: true,
			DefaultMemberPermissions: discord.NewPermissions(
				discord.PermissionAdministrator,
			),
			Options: []discord.CommandOption{
				&discord.ChannelOption{
					OptionName:  "announcement_channel",
					Description: "Channel where the announcement should be sent to",
					Required:    true,
				},
				&discord.StringOption{
					OptionName:  "starts_at",
					Description: fmt.Sprintf("Time when the first announcement starts. Must be in this format: %s", parse.LayoutDateTime),
					MinLength:   option.NewInt(len(parse.LayoutDateTime)),
					MaxLength:   option.NewInt(len(parse.LayoutDateTime)),
					Required:    true,
				},
				&discord.StringOption{
					OptionName:  "ends_at",
					Description: fmt.Sprintf("Time when the announcements should stop. Must be in this format: %s", parse.LayoutDateTime),
					MinLength:   option.NewInt(len(parse.LayoutDateTime)),
					MaxLength:   option.NewInt(len(parse.LayoutDateTime)),
					Required:    true,
				},
				&discord.StringOption{
					OptionName:   "location",
					Description:  "Timzone location, e.g. Europe/Berlin.",
					MinLength:    option.NewInt(1),
					Required:     true,
					Autocomplete: true,
				},
				&discord.StringOption{
					OptionName:  "interval",
					Description: "Interval at and for which the announcement should be sent. e.g. 24h (1 day), 168h (1 week)",
					MinLength:   option.NewInt(2),  // 1h is min
					MaxLength:   option.NewInt(11), // 8760h00m00s is max
					Required:    true,
				},
				&discord.StringOption{
					OptionName:  "custom_text_before",
					Description: "Custom text before the generated annoncement.",
					Required:    false,
				},
				&discord.StringOption{
					OptionName:  "custom_text_after",
					Description: "Custom text after the generated annoncement.",
					Required:    false,
				},
			},
		},
	}

	// update user facing commands
	return cmdroute.OverwriteCommands(b.state, userCommandList)
}
