package options

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
)

func Snowflake(name string, options discord.CommandInteractionOptions) (discord.Snowflake, error) {
	s, err := options.Find(name).SnowflakeValue()
	if err != nil {
		return 0, fmt.Errorf("invalid parameter %q: %w", name, err)
	}
	return s, nil
}

func RoleID(name string, options discord.CommandInteractionOptions) (discord.RoleID, error) {
	s, err := options.Find(name).SnowflakeValue()
	if err != nil {
		return 0, fmt.Errorf("invalid role parameter %q: %w", name, err)
	}
	return discord.RoleID(s), nil
}
