package discordutils

import (
	"errors"

	"github.com/diamondburned/arikawa/v3/utils/httputil"
)

func IsStatus(err error, code int) bool {
	if err == nil {
		return false
	}
	var herr *httputil.HTTPError
	if errors.As(err, &herr) {
		return herr.Status == code
	}

	return false
}
