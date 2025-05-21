package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/jxs13/league-discord-bot/internal/parse"
)

func New() *Config {
	return &Config{
		DSN:                 filepath.Join(filepath.Dir(os.Args[0]), "league.db"),
		BackupInterval:      24 * time.Hour,
		ChannelAccessOffset: 7 * 24 * time.Hour,
		RequirementsOffset:  24 * time.Hour,
		ChannelDeleteOffset: 1 * time.Hour,
		ReminderIntervals:   []time.Duration{24 * time.Hour, 1 * time.Hour, 15 * time.Minute, 5 * time.Minute, 30 * time.Second},
	}
}

type Config struct {
	DSN            string        `koanf:"dsn" description:"database file path (DSN)"`
	DiscordToken   string        `koanf:"discord.token" description:"discord bot token"`
	BackupInterval time.Duration `koanf:"backup.interval" description:"interval for creating backups, e.g. 0s (disabled), 1m, 1h, 12h, 24h, 168h, 720h"`
	BackupDir      string
	BackupFile     string

	ReminderIntervalsString string `koanf:"reminder.intervals" description:"default guild configuration list of reminder intervals to remind players before a match, e.g. 24h,1h,15m,5m,30s"`
	ReminderIntervals       []time.Duration

	ChannelAccessOffset time.Duration `koanf:"guild.channel.access.offset" description:"default time offset for granting access to channels before a match"`
	RequirementsOffset  time.Duration `koanf:"requirements.offset" description:"default time offset for participation requirements to be met before a match"`
	ChannelDeleteOffset time.Duration `koanf:"guild.channel.delete.offset" description:"default time offset for deleting channels after a match"`
}

func (c *Config) Validate() error {

	if c.DiscordToken == "" {
		return fmt.Errorf("discord token is required")
	}

	if c.DSN == "" {
		return errors.New("database DSN is missing")
	}

	err := ValidatableGuildConfig(c.ChannelAccessOffset, c.RequirementsOffset, c.ChannelDeleteOffset)
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

	c.BackupDir = path.Join(path.Dir(u.Path), "backups")
	c.BackupFile = path.Base(u.Path)

	v := u.Query()
	if !v.Has("_txlock") {
		v["_txlock"] = []string{"immediate"} // "deferred" (the default), "immediate", or "exclusive"
	}

	u.RawQuery = v.Encode()
	dsn = u.String()
	c.DSN = dsn
	return nil
}
