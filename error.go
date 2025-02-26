package spool

type yoError interface {
	NotFound() bool
}

func isErrNotFound(err error) bool {
	if err, ok := err.(yoError); ok {
		return err.NotFound()
	}
	return false
}
