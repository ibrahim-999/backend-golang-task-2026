package ports

type PasswordHasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) error
}

type TokenClaims struct {
	UserID uint64
	Role   string
}

type TokenIssuer interface {
	Issue(userID uint64, role string) (string, error)
}

type TokenVerifier interface {
	Verify(token string) (TokenClaims, error)
}
