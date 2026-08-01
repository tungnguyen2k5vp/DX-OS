package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid access token")
	ErrUnknownKey   = errors.New("token signing key is unknown")
)

type VerifierConfig struct {
	Issuer   string
	Audience string
	JWKSURL  string
}

type Principal struct {
	Subject  string   `json:"subject"`
	Username string   `json:"username"`
	Email    string   `json:"email,omitempty"`
	Roles    []string `json:"roles"`
}

type Claims struct {
	jwt.RegisteredClaims
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

type Verifier struct {
	issuer   string
	audience string
	jwksURL  string
	client   *http.Client
	cacheTTL time.Duration

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

func NewVerifier(ctx context.Context, cfg VerifierConfig) (*Verifier, error) {
	verifier := &Verifier{
		issuer:   cfg.Issuer,
		audience: cfg.Audience,
		jwksURL:  cfg.JWKSURL,
		client:   &http.Client{Timeout: 5 * time.Second},
		cacheTTL: 15 * time.Minute,
		keys:     make(map[string]*rsa.PublicKey),
	}
	if err := verifier.refresh(ctx); err != nil {
		return nil, err
	}
	return verifier, nil
}

func (v *Verifier) Verify(ctx context.Context, rawToken string) (Principal, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		rawToken,
		claims,
		func(token *jwt.Token) (any, error) {
			kid, ok := token.Header["kid"].(string)
			if !ok || kid == "" {
				return nil, fmt.Errorf("%w: missing kid", ErrInvalidToken)
			}
			return v.signingKey(ctx, kid)
		},
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(v.issuer),
		jwt.WithAudience(v.audience),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil || !token.Valid || claims.Subject == "" {
		return Principal{}, ErrInvalidToken
	}

	roles := append([]string(nil), claims.RealmAccess.Roles...)
	sort.Strings(roles)
	return Principal{
		Subject:  claims.Subject,
		Username: claims.PreferredUsername,
		Email:    claims.Email,
		Roles:    roles,
	}, nil
}

func (v *Verifier) signingKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key := v.keys[kid]
	stale := time.Since(v.fetchedAt) > v.cacheTTL
	v.mu.RUnlock()
	if key != nil && !stale {
		return key, nil
	}

	if err := v.refresh(ctx); err != nil {
		if key != nil {
			return key, nil
		}
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	if key = v.keys[kid]; key == nil {
		return nil, ErrUnknownKey
	}
	return key, nil
}

type jwksDocument struct {
	Keys []jsonWebKey `json:"keys"`
}

type jsonWebKey struct {
	KeyID     string `json:"kid"`
	KeyType   string `json:"kty"`
	Algorithm string `json:"alg"`
	Use       string `json:"use"`
	Modulus   string `json:"n"`
	Exponent  string `json:"e"`
}

func (v *Verifier) refresh(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("create JWKS request: %w", err)
	}
	response, err := v.client.Do(request)
	if err != nil {
		return fmt.Errorf("fetch JWKS: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch JWKS: unexpected status %d", response.StatusCode)
	}

	var document jwksDocument
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err = decoder.Decode(&document); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, candidate := range document.Keys {
		if candidate.KeyID == "" || candidate.KeyType != "RSA" {
			continue
		}
		if candidate.Algorithm != "" && candidate.Algorithm != "RS256" {
			continue
		}
		key, parseErr := parseRSAKey(candidate.Modulus, candidate.Exponent)
		if parseErr != nil {
			continue
		}
		keys[candidate.KeyID] = key
	}
	if len(keys) == 0 {
		return errors.New("JWKS contains no usable RS256 key")
	}

	v.mu.Lock()
	v.keys = keys
	v.fetchedAt = time.Now()
	v.mu.Unlock()
	return nil
}

func parseRSAKey(encodedModulus, encodedExponent string) (*rsa.PublicKey, error) {
	modulusBytes, err := base64.RawURLEncoding.DecodeString(encodedModulus)
	if err != nil {
		return nil, fmt.Errorf("decode modulus: %w", err)
	}
	exponentBytes, err := base64.RawURLEncoding.DecodeString(encodedExponent)
	if err != nil {
		return nil, fmt.Errorf("decode exponent: %w", err)
	}
	if len(modulusBytes) == 0 || len(exponentBytes) == 0 || len(exponentBytes) > 4 {
		return nil, errors.New("invalid RSA key material")
	}

	exponent := 0
	for _, part := range exponentBytes {
		exponent = exponent<<8 + int(part)
	}
	if exponent < 3 {
		return nil, errors.New("invalid RSA exponent")
	}

	return &rsa.PublicKey{N: new(big.Int).SetBytes(modulusBytes), E: exponent}, nil
}
