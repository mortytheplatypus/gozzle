package log

import (
	"errors"
	"regexp"
)

var (
	ErrInvalidServiceID = errors.New("invalid service id")
	ErrNotRegistered    = errors.New("service not registered")
)

var serviceIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

func ValidateServiceID(id string) error {

	if id == "services" || !serviceIDPattern.MatchString(id) {
		return ErrInvalidServiceID
	}

	return nil
}
