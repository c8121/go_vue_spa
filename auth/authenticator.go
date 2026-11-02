package auth

type Authenticator interface {
	Authenticate(username string, credentials []byte) (*AuthenticatedUser, error)
}
