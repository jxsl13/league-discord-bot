package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func New() *Config {
	return &Config{
		DSN: filepath.Join(filepath.Dir(os.Args[0]), "league.db"),
	}
}

type Config struct {
	DSN          string `koanf:"dsn" description:"database DSN, e.g. ./db.sqlite"`
	DiscordToken string `koanf:"discord.token" description:"discord bot token"`
}

func (c *Config) Validate() error {

	if c.DiscordToken == "" {
		return fmt.Errorf("discord token is required")
	}

	if c.DSN == "" {
		return errors.New("database DSN is missing")
	}

	return nil
}
