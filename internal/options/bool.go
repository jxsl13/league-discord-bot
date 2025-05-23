package options

import (
	"fmt"
	"strconv"

	"github.com/diamondburned/arikawa/v3/discord"
)

func Bool(name string, options discord.CommandInteractionOptions) (bool, error) {
	s := options.Find(name).String()
	b, err := strconv.ParseBool(s)
	if err != nil {
		return false, fmt.Errorf("invalid %q expected format: true, false: %w", name, err)
	}

	return b, nil
}

func BoolInt64(name string, options discord.CommandInteractionOptions) (int64, error) {
	b, err := Bool(name, options)
	if err != nil {
		return 0, err
	}
	if b {
		return 1, nil
	}
	return 0, nil
}

func BoolInt64Option(name string, options discord.CommandInteractionOptions) (int64, bool, error) {
	s := options.Find(name).String()
	if s == "" {
		return 0, false, nil
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return 0, false, fmt.Errorf("invalid %q expected format: true, false: %w", name, err)
	}

	if b {
		return 1, true, nil
	}

	return 0, true, nil
}
