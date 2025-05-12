package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	_ "time/tzdata" // we want to have an embedded timezone database, because we work with those

	"github.com/jxs13/league-discord-bot/bot"
	"github.com/jxs13/league-discord-bot/config"
	"github.com/jxs13/league-discord-bot/migrations"
	"github.com/jxsl13/cli-config-boilerplate/cliconfig"
	"github.com/spf13/cobra"
	"modernc.org/sqlite"
)

func main() {

	err := NewRootCmd().Execute()
	if err != nil {
		os.Exit(1)
	}
}

func NewRootCmd() *cobra.Command {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)

	rootContext := rootContext{
		Context: ctx,
	}

	// cmd represents the run command
	cmd := &cobra.Command{
		Use:   "league-discord-bot",
		Short: "match management bot for the Teeworlds gCTF league. Can also be used for other team based games.",
		RunE:  rootContext.RunE,
		Args:  cobra.ExactArgs(0),
		PostRunE: func(cmd *cobra.Command, args []string) error {
			if rootContext.DB != nil {
				rootContext.DB.Close()
			}
			cancel()
			return nil
		},
	}

	// register flags but defer parsing and validation of the final values
	cmd.PreRunE = rootContext.PreRunE(cmd)

	// register flags but defer parsing and validation of the final values
	cmd.AddCommand(NewCompletionCmd(cmd.Name()))
	return cmd
}

type rootContext struct {
	Context context.Context

	// set in PreRunE
	Config *config.Config
	DB     *sql.DB
}

func (c *rootContext) PreRunE(cmd *cobra.Command) func(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	c.Config = config.New()
	runParser := cliconfig.RegisterFlags(c.Config, true, cmd)
	return func(cmd *cobra.Command, args []string) error {
		err := runParser()
		if err != nil {
			return err
		}

		pragmas := strings.Join(
			[]string{
				"PRAGMA journal_mode=WAL;",
				"PRAGMA encoding = 'UTF-8';",
				"PRAGMA foreign_keys = ON;",
				"PRAGMA busy_timeout = 300000;",
				"PRAGMA synchronous = NORMAL;",
				// "PRAGMA journal_size_limit = 67108864;",
				// "PRAGMA mmap_size = 134217728;",
				// "PRAGMA cache_size = 2000;",
			}, " ")
		sqlite.RegisterConnectionHook(func(conn sqlite.ExecQuerierContext, dsn string) error {

			_, err := conn.ExecContext(c.Context, pragmas, nil)
			if err != nil {
				return fmt.Errorf("failed to set PRAGMA: %w", err)
			}
			return nil
		})

		db, err := sql.Open("sqlite", c.Config.DSN)
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		db.SetMaxIdleConns(10)
		db.SetConnMaxIdleTime(30 * time.Minute)
		db.SetConnMaxLifetime(12 * time.Hour)

		c.DB = db

		err = migrations.Migrate(c.Context, db)
		if err != nil {
			return err
		}

		return nil
	}
}

func (c *rootContext) RunE(cmd *cobra.Command, args []string) error {
	b, err := bot.New(
		c.Context,
		c.Config.DiscordToken,
		c.DB,
		c.Config.ReminderIntervals,
		c.Config.BackoffMinDuration,
		c.Config.AsyncLoopInterval,
		c.Config.ChannelAccessOffset,
		c.Config.RequirementsOffset,
		c.Config.ChannelDeleteOffset,
	)
	if err != nil {
		return err
	}
	defer b.Close()

	log.Println("starting bot")
	err = b.Connect(c.Context)
	if err != nil {
		return err
	}
	return nil
}
