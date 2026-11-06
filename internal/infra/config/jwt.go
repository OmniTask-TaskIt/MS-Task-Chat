package config

import (
	"errors"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrJWTSecretMissing = errors.New("JWT secret is not configured")
	ErrInvalidJWT       = errors.New("invalid authentication token")
)

type JWTValidator struct {
	secret []byte
}

func NewJWTValidatorFromEnv() *JWTValidator {
	return NewJWTValidator(envOrDefault("JWT_SECRET", ""))
}

func NewJWTValidator(secret string) *JWTValidator {
	return &JWTValidator{secret: []byte(strings.TrimSpace(secret))}
}

func (v *JWTValidator) Validate(tokenString string) (string, error) {
	if len(v.secret) == 0 {
		return "", ErrJWTSecretMissing
	}
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return "", ErrInvalidJWT
	}

	signingMethod := signingMethodForSecret(v.secret)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != signingMethod.Alg() {
			return nil, ErrInvalidJWT
		}
		return v.secret, nil
	}, jwt.WithValidMethods([]string{signingMethod.Alg()}), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return "", ErrInvalidJWT
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", ErrInvalidJWT
	}

	if role, ok := claims["role"].(string); !ok || strings.TrimSpace(role) == "" {
		return "", ErrInvalidJWT
	}

	for _, claimName := range []string{"userId", "id", "sub"} {
		if identity := claimString(claims[claimName]); identity != "" {
			return identity, nil
		}
	}
	return "", ErrInvalidJWT
}

func signingMethodForSecret(secret []byte) jwt.SigningMethod {
	switch {
	case len(secret) >= 64:
		return jwt.SigningMethodHS512
	case len(secret) >= 48:
		return jwt.SigningMethodHS384
	default:
		return jwt.SigningMethodHS256
	}
}

func claimString(value interface{}) string {
	switch value := value.(type) {
	case string:
		return strings.TrimSpace(value)
	case float64:
		return strings.TrimSpace(strconv.FormatFloat(value, 'f', -1, 64))
	default:
		return ""
	}
}
