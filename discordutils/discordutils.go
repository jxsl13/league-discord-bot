package discordutils

import (
	"fmt"
	"time"
)

func ToDiscordTimestamp(t time.Time) string {
	return fmt.Sprintf("<t:%d:F>", t.Unix())
}
