package auth

import (
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type JWTService struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

func NewJWTService(secret, issuer string, ttl time.Duration) *JWTService {
	return &JWTService{secret: []byte(secret), issuer: issuer, ttl: ttl}
}

func (s *JWTService) Issue(userID uint64, role string) (string, error) {
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"sub":  strconv.FormatUint(userID, 10),
		"role": role,
		"iss":  s.issuer,
		"iat":  jwt.NewNumericDate(now),
		"exp":  jwt.NewNumericDate(now.Add(s.ttl)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", errs.Internal("token_sign_failed", "failed to sign token")
	}
	return signed, nil
}

func (s *JWTService) Verify(token string) (ports.TokenClaims, error) {
	parsed, err := jwt.Parse(
		token,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errs.Unauthorized("invalid_token", "invalid token")
			}
			return s.secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(s.issuer),
	)
	if err != nil || !parsed.Valid {
		return ports.TokenClaims{}, errs.Unauthorized("invalid_token", "invalid or expired token")
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return ports.TokenClaims{}, errs.Unauthorized("invalid_token", "invalid token claims")
	}

	subject, ok := claims["sub"].(string)
	if !ok {
		return ports.TokenClaims{}, errs.Unauthorized("invalid_token", "invalid token subject")
	}
	userID, parseErr := strconv.ParseUint(subject, 10, 64)
	if parseErr != nil {
		return ports.TokenClaims{}, errs.Unauthorized("invalid_token", "invalid token subject")
	}

	role, _ := claims["role"].(string)

	return ports.TokenClaims{UserID: userID, Role: role}, nil
}

var _ ports.TokenIssuer = (*JWTService)(nil)
var _ ports.TokenVerifier = (*JWTService)(nil)
