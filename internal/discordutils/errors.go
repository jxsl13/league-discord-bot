package discordutils

import (
	"errors"
	"slices"

	"github.com/diamondburned/arikawa/v3/utils/httputil"
)

func IsStatus(err error, codes ...int) bool {
	if err == nil {
		return false
	}
	var herr *httputil.HTTPError
	if errors.As(err, &herr) {
		return slices.Contains(codes, herr.Status)
	}

	return false
}

func IsStatus4XX(err error) bool {
	if err == nil {
		return false
	}
	var herr *httputil.HTTPError
	if errors.As(err, &herr) {
		return herr.Status/100 == 4
	}

	return false
}
