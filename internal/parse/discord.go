package parse

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
)

func Snowflake(id string) (discord.Snowflake, error) {
	s, err := discord.ParseSnowflake(id)
	if err != nil {
		return 0, fmt.Errorf("invalid snowflake: %s: %w", id, err)
	}
	return s, nil
}

func GuildID(id string) (discord.GuildID, error) {
	s, err := Snowflake(id)
	if err != nil {
		return 0, fmt.Errorf("failed to parse guild ID: %w", err)
	}
	return discord.GuildID(s), nil
}

func ChannelID(id string) (discord.ChannelID, error) {
	s, err := Snowflake(id)
	if err != nil {
		return 0, fmt.Errorf("failed to parse channel ID: %w", err)
	}
	return discord.ChannelID(s), nil
}

func MessageID(id string) (discord.MessageID, error) {
	s, err := Snowflake(id)
	if err != nil {
		return 0, fmt.Errorf("failed to parse message ID: %w", err)
	}
	return discord.MessageID(s), nil
}

func RoleID(id string) (discord.RoleID, error) {
	s, err := Snowflake(id)
	if err != nil {
		return 0, fmt.Errorf("failed to parse role ID: %w", err)
	}
	return discord.RoleID(s), nil
}

func UserID(id string) (discord.UserID, error) {
	s, err := Snowflake(id)
	if err != nil {
		return 0, fmt.Errorf("failed to parse user ID: %w", err)
	}
	return discord.UserID(s), nil
}
