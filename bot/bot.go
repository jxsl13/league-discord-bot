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
	"github.com/jxs13/league-discord-bot/internal/parse"
	"github.com/jxs13/league-discord-bot/internal/reminder"
	"github.com/jxs13/league-discord-bot/internal/timerutils"
	"github.com/jxs13/league-discord-bot/sqlc"
)

var userCommandList = []api.CreateCommandData{
	{
		Name:           "configure",
		Description:    "Configure the bot for the current guild",
		NoDMPermission: true,
		DefaultMemberPermissions: discord.NewPermissions(
			discord.PermissionAdministrator,
		),
		Options: []discord.CommandOption{
			&discord.StringOption{
				OptionName:  "channel_delete_offset",
				Description: "Duration after match until channel deletion, at least 1h.",
				MinLength:   option.NewInt(2),
				MaxLength:   option.NewInt(11),
				Required:    true,
			},
			&discord.StringOption{
				OptionName:  "channel_access_offset",
				Description: "Duration after match until channel deletion, at least 1h.",
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
				Description: fmt.Sprintf("Time when the match starts. Must be in this format: %s", parse.LayoutDateTimeWithZone),
				MinLength:   option.NewInt(len(parse.LayoutDateTimeWithZone)),
				MaxLength:   option.NewInt(len(parse.LayoutDateTimeWithZone)),
				Required:    true,
			},
			&discord.StringOption{
				OptionName:  "location",
				Description: "Timzone location, e.g. Europe/Berlin.",
				MinLength:   option.NewInt(1),
				Required:    true,
			},
			&discord.IntegerOption{
				OptionName:  "participants_per_team",
				Description: "Number of participants per team. (3vs3 -> 3)",
				Min:         option.NewInt(1),
				Required:    true,
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
}

type Bot struct {
	ctx    context.Context
	state  *state.State
	db     *sql.DB
	userID discord.UserID
	wg     *sync.WaitGroup

	reminder                   *reminder.Reminder
	defaultChannelAccessOffset time.Duration
	defaultChannelDeleteOffset time.Duration
}

// New requires a discord bot token and returns a Bot instance.
// A bot token starts with Nj... and can be obtained from the discord developer portal.
func New(
	ctx context.Context,
	token string,
	db *sql.DB,
	reminder *reminder.Reminder,
	minBackoff time.Duration,
	loopInterval time.Duration,
	defaultChannelAccessOffset time.Duration,
	defaultChannelDeleteOffset time.Duration,
) (*Bot, error) {

	s := state.New("Bot " + token)

	bot := &Bot{
		ctx:                        ctx,
		state:                      s,
		db:                         db,
		wg:                         &sync.WaitGroup{},
		reminder:                   reminder,
		defaultChannelAccessOffset: defaultChannelAccessOffset,
		defaultChannelDeleteOffset: defaultChannelDeleteOffset,
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
				bot.asyncGiveChannelAccess,
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
				bot.asyncReminder,
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
				bot.asyncDeleteExpiredMatchChannel,
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

	r := cmdroute.NewRouter()

	// admin commands
	r.AddFunc("configure", bot.commandGuildConfigure)

	// admin + user commands
	r.AddFunc("schedule-match", bot.commandScheduleMatch)
	r.AddFunc("reschedule-match", bot.commandRescheduleMatch)

	s.AddInteractionHandler(r)

	// update user facing commands
	err := cmdroute.OverwriteCommands(s, userCommandList)
	if err != nil {
		return nil, err
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

func (b *Bot) Queries(ctx context.Context) (q *sqlc.Queries, err error) {
	conn, err := b.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	return sqlc.New(conn), nil
}
