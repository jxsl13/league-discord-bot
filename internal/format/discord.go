package format

import (
	"fmt"
	"time"
)

func DiscordTimestamp(t time.Time) string {
	return fmt.Sprintf("<t:%d:F>", t.UTC().Unix())
}
