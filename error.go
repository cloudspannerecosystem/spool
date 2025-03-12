package spool

import "errors"

type yoError interface {
	NotFound() bool
}

func isErrNotFound(err error) bool {
	var yErr yoError
	if errors.As(err, &yErr) {
		return yErr.NotFound()
	}
	return false
}
