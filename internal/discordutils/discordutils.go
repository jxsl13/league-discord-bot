package discordutils

import (
	"fmt"
	"time"
)

func Timestamp(t time.Time) string {
	return fmt.Sprintf("<t:%d:F>", t.UTC().Unix())
}
