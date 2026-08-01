package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestVerifierAcceptsValidKeycloakStyleToken(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{{
				"kid": "test-key",
				"kty": "RSA",
				"alg": "RS256",
				"use": "sig",
				"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
			}},
		})
	}))
	defer server.Close()

	verifier, err := NewVerifier(context.Background(), VerifierConfig{
		Issuer:   "http://issuer.test/realms/dx-os",
		Audience: "dx-api",
		JWKSURL:  server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-123",
			Issuer:    "http://issuer.test/realms/dx-os",
			Audience:  jwt.ClaimStrings{"dx-api"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		PreferredUsername: "employee.demo",
		Email:             "employee.demo@example.test",
	}
	claims.RealmAccess.Roles = []string{"employee"}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key"
	rawToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatal(err)
	}

	principal, err := verifier.Verify(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("Verify() unexpected error: %v", err)
	}
	if principal.Subject != "user-123" || principal.Username != "employee.demo" {
		t.Fatalf("unexpected principal: %#v", principal)
	}
}

func TestVerifierRejectsWrongAudience(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{{
				"kid": "test-key",
				"kty": "RSA",
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
			}},
		})
	}))
	defer server.Close()

	verifier, err := NewVerifier(context.Background(), VerifierConfig{
		Issuer:   "http://issuer.test/realms/dx-os",
		Audience: "dx-api",
		JWKSURL:  server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	claims := Claims{RegisteredClaims: jwt.RegisteredClaims{
		Subject:   "user-123",
		Issuer:    "http://issuer.test/realms/dx-os",
		Audience:  jwt.ClaimStrings{"another-api"},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
	}}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key"
	rawToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = verifier.Verify(context.Background(), rawToken); err == nil {
		t.Fatal("Verify() expected wrong audience to be rejected")
	}
}
