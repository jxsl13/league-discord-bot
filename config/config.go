package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/jxs13/league-discord-bot/internal/parse"
)

func New() *Config {
	return &Config{
		DSN:                        filepath.Join(filepath.Dir(os.Args[0]), "league.db"),
		ChannelAccessOffset:        7 * 24 * time.Hour,
		ParticipationConfirmOffset: 24 * time.Hour,
		ChannelDeleteOffset:        24 * time.Hour,

		AsyncLoopInterval:  15 * time.Second,
		BackoffMinDuration: 5 * time.Second,
		ReminderIntervals:  []time.Duration{24 * time.Hour, 1 * time.Hour, 15 * time.Minute, 5 * time.Minute, 30 * time.Second},
	}
}

type Config struct {
	DSN          string `koanf:"dsn" description:"database file path (DSN)"`
	DiscordToken string `koanf:"discord.token" description:"discord bot token"`

	AsyncLoopInterval  time.Duration `koanf:"async.loop.interval" description:"interval for async loops, should be a small value e.g. 10s, 30s, 1m"`
	BackoffMinDuration time.Duration `koanf:"backoff.min.duration" description:"minimum duration for backoff upon api error, must be smaller than async loop interval e.g. 10s, 30s, 1m"`

	ReminderIntervalsString string `koanf:"reminder.intervals" description:"default guild configuration list of reminder intervals to remind players before a match, e.g. 24h,1h,15m,5m,30s"`
	ReminderIntervals       []time.Duration

	ChannelAccessOffset        time.Duration `koanf:"guild.channel.access.offset" description:"default time offset for granting access to channels before a match"`
	ParticipationConfirmOffset time.Duration `koanf:"participation.confirm.offset" description:"default time offset for confirming participation before a match"`
	ChannelDeleteOffset        time.Duration `koanf:"guild.channel.delete.offset" description:"default time offset for deleting channels after a match"`
}

func (c *Config) Validate() error {

	if c.DiscordToken == "" {
		return fmt.Errorf("discord token is required")
	}

	if c.DSN == "" {
		return errors.New("database DSN is missing")
	}

	if c.AsyncLoopInterval < 1*time.Second {
		return fmt.Errorf("async loop interval must be greater or equal to 1s, e.g. 10s, 30s, 1m")
	}

	if c.AsyncLoopInterval > 5*time.Minute {
		return fmt.Errorf("async loop interval must be smaller than or equal to 5m, e.g. 10s, 30s, 1m")
	}

	if c.BackoffMinDuration < 1*time.Second {
		return fmt.Errorf("backoff min duration must be greater or equal to 1s, e.g. 5s, 10s, 30s, 1m")
	}

	if c.BackoffMinDuration > 5*time.Minute {
		return fmt.Errorf("backoff min duration must be smaller than or equal to 5m, e.g. 5s, 10s, 30s, 1m")
	}

	if c.BackoffMinDuration > c.AsyncLoopInterval {
		return fmt.Errorf("backoff min duration must be smaller than or equal to async loop interval, e.g. 10s, 30s, 1m")
	}

	err := ValidatableGuildConfig(c.ChannelAccessOffset, c.ParticipationConfirmOffset, c.ChannelDeleteOffset)
	if err != nil {
		return err
	}

	intervals, err := parse.ReminderIntervals(c.ReminderIntervalsString)
	if err != nil {
		return err
	}

	for _, d := range intervals {
		if d > c.ChannelAccessOffset {
			return fmt.Errorf("reminder interval must be smaller than or equal to channel access offset, e.g. 24h, 1h30m: it does not make sense to notify users before they can access the match channel: interval: %s, access offset: %s", d, c.ChannelAccessOffset)
		}
	}

	dsn := filepath.ToSlash(c.DSN)
	u, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("error parsing DSN: %w", err)
	}

	v := u.Query()
	if !v.Has("_txlock") {
		v["_txlock"] = []string{"immediate"} // "deferred" (the default), "immediate", or "exclusive"
	}

	u.RawQuery = v.Encode()
	dsn = u.String()
	c.DSN = dsn
	return nil
}
