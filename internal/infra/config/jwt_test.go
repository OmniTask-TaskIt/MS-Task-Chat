package config

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testJWTSecret = "01234567890123456789012345678901"

func TestJWTValidatorAcceptsAccessTokenSubject(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "sasupu08@gmail.com",
		"role": "USER",
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatal(err)
	}

	userID, err := NewJWTValidator(testJWTSecret).Validate(signed)
	if err != nil {
		t.Fatal(err)
	}
	if userID != "sasupu08@gmail.com" {
		t.Fatalf("user ID = %q, want %q", userID, "sasupu08@gmail.com")
	}
}

func TestJWTValidatorRejectsInvalidToken(t *testing.T) {
	if _, err := NewJWTValidator(testJWTSecret).Validate("not-a-jwt"); err != ErrInvalidJWT {
		t.Fatalf("error = %v, want %v", err, ErrInvalidJWT)
	}
}
