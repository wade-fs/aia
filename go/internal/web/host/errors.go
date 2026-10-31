package host

import "errors"

var (
	ErrInvalidToken = errors.New("invalid or already used login token")
	ErrExpiredToken = errors.New("login token has expired")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)
