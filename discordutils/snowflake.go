package discordutils

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

func ParseChannelID(id string) (discord.ChannelID, error) {
	s, err := Snowflake(id)
	if err != nil {
		return 0, fmt.Errorf("failed to parse channel ID: %w", err)
	}
	return discord.ChannelID(s), nil
}

func ParseRoleID(id string) (discord.RoleID, error) {
	s, err := Snowflake(id)
	if err != nil {
		return 0, fmt.Errorf("failed to parse role ID: %w", err)
	}
	return discord.RoleID(s), nil
}

func ParseUserID(id string) (discord.UserID, error) {
	s, err := Snowflake(id)
	if err != nil {
		return 0, fmt.Errorf("failed to parse user ID: %w", err)
	}
	return discord.UserID(s), nil
}
