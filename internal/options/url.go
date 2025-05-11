package options

import (
	"fmt"
	"net/url"

	"github.com/diamondburned/arikawa/v3/discord"
)

func OptionalUrl(name string, options discord.CommandInteractionOptions) (string, bool, error) {
	o := options.Find(name)
	if o.Type == 0 {
		return "", false, nil
	}
	s := o.String()
	url, err := url.ParseRequestURI(s)
	if err != nil {
		return "", false, fmt.Errorf("invalid url parameter %q: %w", name, err)
	}

	return url.String(), true, nil
}
