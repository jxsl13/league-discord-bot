package discordutils

import "github.com/diamondburned/arikawa/v3/discord"

func LastChannelPosition(channels []discord.Channel) int {
	var lastPos int
	for _, channel := range channels {
		if channel.Position > lastPos {
			lastPos = channel.Position
		}
	}
	return lastPos + 1
}
