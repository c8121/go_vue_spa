package auth

var (
	authenticator Authenticator
)

func SetAuthenticator(auth Authenticator) {
	authenticator = auth
}

func GetAuthenticator() Authenticator {
	return authenticator
}
