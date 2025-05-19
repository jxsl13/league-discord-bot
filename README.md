# League Discord Bot

This is a match making discord bot with a small built-in sqlite3 database and the ability to schedule new league matches for teams.
There are by default two teams that can be set, each via their role and there are two additional individuals that can be set, the moderator and an optional streamer with a corresponding, but also optional, streaming url, which can be any valid url.

Initially the bot creates a category under which he creates new channels that are only visible by him and after some time also visible by the scheduled moderator and the streamer as well as all clan members of the two roles.
By default participants can see the channel up to 7 days in advance.

The bot requests up to N players to confirm their participation from each participating team.
Once enough plalyers are met and a specific point in time is reached, the participation confirmation period is over and everyone in the channel is notified that the match's planning can be finalized.

The bot starts to remind all participants (teams, moderator and streamer) of their match multiple times, up until when the actual match starts.
By default, the bot deletes the match channel after 24 hours affter the scheduled game.

In order to install the bot on your server, you can use this link:

[Bot Invite link](https://discord.com/oauth2/authorize?client_id=1370994546799804426&permissions=1707535384444016&integration_type=0&scope=bot)

The bot is capable of running on multiple Discord servers, which is why it is not necessary to create a bot for each server and it is also not necessary to host your own instance of the bot.

## Installation

You can use either docker or docker compose in order to run the container image of the bot or you run it directly by downloading it from the release page.

The bot is a selfcontained single binary and does not require any additional dependencies, other than knowledge about what target operating system the bot is running on, in order to download the correct executable.

## Configuratiion

The bot can be configured via three ways:

1. Environment variables
2. `.env` Config file
3. Command line flags `--help` for more information.

```shell
$ league-discord-bot --help
Environment variables:
  DSN                            database file path (DSN) (default: "league.db")
  DISCORD_TOKEN                  discord bot token
  BACKUP_INTERVAL                interval for creating backups, e.g. 0s (disabled), 1m, 1h, 12h, 24h, 168h, 720h (default: "24h0m0s")
  ASYNC_LOOP_INTERVAL            interval for async loops, should be a small value e.g. 10s, 30s, 1m (default: "15s")
  REMINDER_INTERVALS             default guild configuration list of reminder intervals to remind players before a match, e.g. 24h,1h,15m,5m,30s
  GUILD_CHANNEL_ACCESS_OFFSET    default time offset for granting access to channels before a match (default: "168h0m0s")
  REQUIREMENTS_OFFSET            default time offset for participation requirements to be met before a match (default: "24h0m0s")
  GUILD_CHANNEL_DELETE_OFFSET    default time offset for deleting channels after a match (default: "1h0m0s")

Usage:
  league-discord-bot [flags]
  league-discord-bot [command]

Available Commands:
  completion  Generate completion script
  help        Help about any command

Flags:
      --async-loop-interval duration           interval for async loops, should be a small value e.g. 10s, 30s, 1m (default 15s)
      --backup-interval duration               interval for creating backups, e.g. 0s (disabled), 1m, 1h, 12h, 24h, 168h, 720h (default 24h0m0s)
  -c, --config string                          .env config file path (or via env variable CONFIG)
      --discord-token string                   discord bot token
      --dsn string                             database file path (DSN) (default "league.db")
      --guild-channel-access-offset duration   default time offset for granting access to channels before a match (default 168h0m0s)
      --guild-channel-delete-offset duration   default time offset for deleting channels after a match (default 1h0m0s)
  -h, --help                                   help for league-discord-bot
      --reminder-intervals string              default guild configuration list of reminder intervals to remind players before a match, e.g. 24h,1h,15m,5m,30s
      --requirements-offset duration           default time offset for participation requirements to be met before a match (default 24h0m0s)

Use "league-discord-bot [command] --help" for more information about a command.
```

