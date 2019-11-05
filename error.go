package spool

type yoError interface {
	NotFound() bool
}

func isErrNotFound(err error) bool {
	switch v := err.(type) {
	case yoError:
		return v.NotFound()
	}
	return false
}
