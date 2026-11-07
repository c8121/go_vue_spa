package auth

import "errors"

var (
	ErrAuthRequired = errors.New("authentication required")
	ErrNoSuchUser   = errors.New("no such user")
	ErrAuthFailed   = errors.New("auth failed")
)

type Authenticator interface {
	Authenticate(username string, credentials []byte) (*AuthenticatedUser, error)
}
