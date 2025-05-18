package format

import (
	"fmt"
	"time"
)

// https://gist.github.com/LeviSnoot/d9147767abeef2f770e9ddcd91eb85aa
func DiscordLongDateTime(t time.Time) string {
	return fmt.Sprintf("<t:%d:F>", t.UTC().Unix())
}

// <t:1543392060:D>
func DiscordLongDate(t time.Time) string {
	return fmt.Sprintf("<t:%d:D>", t.UTC().Unix())
}
