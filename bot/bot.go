package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/jxs13/league-discord-bot/internal/format"
	"github.com/jxs13/league-discord-bot/internal/parse"
	"github.com/jxs13/league-discord-bot/internal/timerutils"
	"github.com/jxs13/league-discord-bot/sqlc"
)

type Bot struct {
	ctx    context.Context
	state  *state.State
	db     *sql.DB
	userID discord.UserID
	wg     *sync.WaitGroup

	defaultNotificationOffsets []time.Duration
	defaultChannelAccessOffset time.Duration
	defaulRequirementsOffset   time.Duration
	defaultChannelDeleteOffset time.Duration

	loopInterval time.Duration
	minBackoff   time.Duration
}

// New requires a discord bot token and returns a Bot instance.
// A bot token starts with Nj... and can be obtained from the discord developer portal.
func New(
	ctx context.Context,
	token string,
	db *sql.DB,
	defaultNotificationOffsets []time.Duration,
	minBackoff time.Duration,
	loopInterval time.Duration,
	defaultChannelAccessOffset time.Duration,
	defaulRequirementsOffset time.Duration,
	defaultChannelDeleteOffset time.Duration,
) (*Bot, error) {

	s := state.New("Bot " + token)
	bot := &Bot{
		ctx:                        ctx,
		state:                      s,
		db:                         db,
		wg:                         &sync.WaitGroup{},
		defaultNotificationOffsets: defaultNotificationOffsets,
		defaultChannelAccessOffset: defaultChannelAccessOffset,
		defaulRequirementsOffset:   defaulRequirementsOffset,
		defaultChannelDeleteOffset: defaultChannelDeleteOffset,
		loopInterval:               loopInterval,
		minBackoff:                 minBackoff,
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
				log.Fatalf("failed to get bot user: %v", err)
			}
			bot.userID = me.ID

			bot.wg.Add(1)
			go timerutils.Loop(
				bot.ctx,
				minBackoff,
				loopInterval,
				bot.asyncGrantChannelAccess,
				func() {
					log.Println("channel access routine stopped")
					bot.wg.Done()
				},
			)

			bot.wg.Add(1)
			go timerutils.Loop(
				bot.ctx,
				minBackoff,
				loopInterval,
				bot.asyncNotifications,
				func() {
					log.Println("reminder routine stopped")
					bot.wg.Done()
				},
			)

			bot.wg.Add(1)
			go timerutils.Loop(
				bot.ctx,
				minBackoff,
				loopInterval,
				bot.asyncDeleteExpiredChannels,
				func() {
					log.Println("channel delete routine stopped")
					bot.wg.Done()
				},
			)

			bot.wg.Add(1)
			go timerutils.Loop(
				bot.ctx,
				minBackoff,
				loopInterval,
				bot.asyncCheckParticipationDeadline,
				func() {
					log.Println("participant deadline check routine stopped")
					bot.wg.Done()
				},
			)

			log.Println("bot is ready")
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

	// admin commands
	r.AddFunc("configure", bot.commandGuildConfigure)
	r.AddFunc("configuration", bot.commandGuildConfiguration)

	// admin + user commands
	r.AddFunc("schedule-match", bot.commandScheduleMatch)
	// r.AddFunc("reschedule-match", bot.commandRescheduleMatch)

	r.AddFunc("notification-list", bot.commandNotificationsList)
	r.AddFunc("notification-delete", bot.commandNotificationsDelete)
	r.AddFunc("notification-add", bot.commandNotificationsAdd)
	// TODO: timezone auto completion: r.AddAutocompleterFunc()

	s.AddInteractionHandler(r)

	err := bot.overrideCommands()
	if err != nil {
		return nil, fmt.Errorf("failed to override commands: %w", err)
	}

	return bot, nil
}

func (b *Bot) Connect(ctx context.Context) error {
	return b.state.Connect(ctx)
}

func (b *Bot) Close() error {
	defer b.wg.Wait()

	return errors.Join(
		b.state.Close(),
	)
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
		err = errors.Join(err, tx.Rollback())
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
				&discord.StringOption{
					OptionName:  "channel_access_offset",
					Description: "How long before the match the user can access the mach channel",
					MinLength:   option.NewInt(2),
					MaxLength:   option.NewInt(11),
					Required:    true,
				},
				&discord.BooleanOption{
					OptionName:  "event_creation_enabled",
					Description: "Automatically reate scheduled events for matches that have a streamer and stream_url",
					Required:    true,
				},
				&discord.StringOption{
					OptionName:  "notification_offsets",
					Description: "Intervals at which to remind before a match e.g. 24h,1h,15m,5m,30s or empty for no defaults",
					Required:    true,
				},
				&discord.StringOption{
					OptionName:  "requirements_offset",
					Description: "Time before the match until which the participation requirements need to be met",
					MinLength:   option.NewInt(2),
					MaxLength:   option.NewInt(11),
					Required:    true,
				},
				&discord.StringOption{
					OptionName:  "channel_delete_offset",
					Description: "Deadline after the match at which the channel is deleted",
					MinLength:   option.NewInt(2),
					MaxLength:   option.NewInt(11),
					Required:    true,
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
	}

	// update user facing commands
	return cmdroute.OverwriteCommands(b.state, userCommandList)
}
