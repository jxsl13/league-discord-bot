package options

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
)

func IntegerOption(name string, options discord.CommandInteractionOptions) (int64, error) {
	i, err := options.Find(name).IntValue()
	if err != nil {
		return 0, fmt.Errorf("invalid integer parameter %q: %w", name, err)
	}
	return i, nil
}

func NotNegativeInteger(name string, options discord.CommandInteractionOptions) (int64, error) {
	i, err := IntegerOption(name, options)
	if err != nil {
		return 0, err
	}
	if i < 0 {
		return 0, fmt.Errorf("invalid integer parameter %q: must be non-negative", name)
	}
	return i, nil
}

func MinInteger(name string, options discord.CommandInteractionOptions, min int64) (int64, error) {
	i, err := IntegerOption(name, options)
	if err != nil {
		return 0, err
	}
	if i < min {
		return 0, fmt.Errorf("invalid integer parameter %q: must be at least %d", name, min)
	}
	return i, nil
}

func MinMaxInteger(name string, options discord.CommandInteractionOptions, min, max int64) (int64, error) {
	i, err := IntegerOption(name, options)
	if err != nil {
		return 0, err
	}
	if i < min {
		return 0, fmt.Errorf("invalid integer parameter %q: must be at least %d", name, min)
	}
	if i > max {
		return 0, fmt.Errorf("invalid integer parameter %q: must be at most %d", name, max)
	}
	return i, nil
}
