package config

import (
	"fmt"
	"time"
)

func ValidatableGuildConfig(access, confirm, delete time.Duration) error {
	if access < 0 {
		return fmt.Errorf("channel access offset must be greater or equal to 0s, e.g. 24h or 1h30m")
	}

	if confirm < 0 {
		return fmt.Errorf("participation confirmation offset must be greater or equal to 0s, e.g. 24h or 1h30m")
	}

	if delete < 0 {
		return fmt.Errorf("channel delete offset must be greater or equal to 0s, e.g. 24h or 1h30m")
	}

	if confirm >= access {
		return fmt.Errorf("participation confirmation offset must be smaller than channel access offset, e.g. 24h, 1h30m: it does not make sense to have confirmed participation before the users can access the match channel: confirmation offset: %s, access offset: %s", confirm, access)
	}

	return nil
}
