package discordutils

import "github.com/diamondburned/arikawa/v3/discord"

func Snowflake(id string) (discord.Snowflake, error) {
	s, err := discord.ParseSnowflake(id)
	if err != nil {
		return 0, err
	}
	return s, nil
}

func ParseChannelID(id string) (discord.ChannelID, error) {
	s, err := Snowflake(id)
	if err != nil {
		return 0, err
	}
	return discord.ChannelID(s), nil
}
