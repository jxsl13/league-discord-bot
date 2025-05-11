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

func UserID(name string, options discord.CommandInteractionOptions) (discord.UserID, error) {
	s, err := options.Find(name).SnowflakeValue()
	if err != nil {
		return 0, fmt.Errorf("invalid role parameter %q: %w", name, err)
	}
	return discord.UserID(s), nil
}

func ChannelID(name string, options discord.CommandInteractionOptions) (discord.ChannelID, error) {
	s, err := options.Find(name).SnowflakeValue()
	if err != nil {
		return 0, fmt.Errorf("invalid role parameter %q: %w", name, err)
	}
	return discord.ChannelID(s), nil
}

func OptionalUserID(name string, options discord.CommandInteractionOptions) (_ discord.UserID, ok bool, err error) {
	o := options.Find(name)
	if o.Type == 0 {
		return 0, false, nil
	}
	s, err := o.SnowflakeValue()
	if err != nil {
		return 0, false, fmt.Errorf("invalid user parameter %q: %w", name, err)
	}
	return discord.UserID(s), true, nil
}
