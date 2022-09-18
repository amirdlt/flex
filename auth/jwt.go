package auth

type JwtAuthenticator[T any] interface {
	Verify(token string) (t T, err error)
	Generate(t T) (token string, err error)
}
