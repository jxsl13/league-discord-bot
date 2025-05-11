package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jxs13/league-discord-bot/reminder"
)

func New() *Config {
	return &Config{
		DSN:                 filepath.Join(filepath.Dir(os.Args[0]), "league.db?_txlock=immediate"),
		ChannelDeleteOffset: 24 * time.Hour,
		ChannelAccessOffset: 7 * 24 * time.Hour,
		AsyncLoopInterval:   15 * time.Second,
		BackoffMinDuration:  5 * time.Second,
		ReminderIntervals:   []time.Duration{24 * time.Hour, 1 * time.Hour, 15 * time.Minute, 5 * time.Minute, 30 * time.Second},
	}
}

type Config struct {
	DSN          string `koanf:"dsn" description:"database file path (DSN)"`
	DiscordToken string `koanf:"discord.token" description:"discord bot token"`

	ChannelDeleteOffset time.Duration `koanf:"guild.channel.delete.offset" description:"default time offset for deleting channels after a match"`
	ChannelAccessOffset time.Duration `koanf:"guild.channel.access.offset" description:"default time offset for granting access to channels after a match"`
	AsyncLoopInterval   time.Duration `koanf:"async.loop.interval" description:"interval for async loops, should be a small value e.g. 10s, 30s, 1m"`
	BackoffMinDuration  time.Duration `koanf:"backoff.min.duration" description:"minimum duration for backoff upon api error, must be smaller than async loop interval e.g. 10s, 30s, 1m"`

	ReminderIntervalsString string `koanf:"reminder.intervals" description:"list of reminder intervals to remind players before a match, e.g. 24h,1h,15m,5m,30s"`
	ReminderIntervals       []time.Duration
}

func (c *Config) Validate() error {

	if c.DiscordToken == "" {
		return fmt.Errorf("discord token is required")
	}

	if c.DSN == "" {
		return errors.New("database DSN is missing")
	}

	if c.ChannelAccessOffset < 0 {
		return fmt.Errorf("channel access offset must be greater or equal to 0s, e.g. 24h or 1h30m")
	}

	if c.ChannelDeleteOffset < 0 {
		return fmt.Errorf("channel delete offset must be greater or equal to 0s, e.g. 24h or 1h30m")
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

	intervals := strings.Split(c.ReminderIntervalsString, ",")
	if len(intervals) == 0 {
		return fmt.Errorf("reminder intervals must be a comma separated list of durations containing at least a single duration, e.g. 24h,1h,15m,5m,30s")
	}

	for _, rs := range intervals {
		rs = strings.TrimSpace(rs)
		d, err := time.ParseDuration(rs)
		if err != nil {
			return fmt.Errorf("error parsing reminder interval %s: %w", rs, err)
		}

		if d < time.Second {
			return fmt.Errorf("reminder interval must be greater or equal to 1s, e.g. 5s, 5m, 15m, 1h, 24h")
		}

		if d > c.ChannelAccessOffset {
			return fmt.Errorf("reminder interval must be smaller than or equal to channel access offset, e.g. 24h, 1h30m: it does not make sense to notify users before they can access the match channel: interval: %s, access offset: %s", d, c.ChannelAccessOffset)
		}

		c.ReminderIntervals = append(c.ReminderIntervals, d)
	}

	dsn := filepath.ToSlash(c.DSN)

	u, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("error parsing DSN: %w", err)
	}

	v := u.Query()
	v["_txlock"] = []string{"immediate"} // "deferred" (the default), "immediate", or "exclusive"

	u.RawQuery = v.Encode()

	c.DSN = u.String()
	return nil
}

func (c *Config) Reminder() *reminder.Reminder {
	r, err := reminder.New(c.ReminderIntervals...)
	if err != nil {
		panic(fmt.Errorf("error creating reminder: %w", err))
	}
	return r
}
