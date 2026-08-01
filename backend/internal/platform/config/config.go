package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Environment       string
	HTTPAddress       string
	DatabaseURL       string
	OIDCIssuer        string
	OIDCJWKSURL       string
	OIDCAudience      string
	AllowedOrigin     string
	NextcloudURL      string
	NextcloudUsername string
	NextcloudPassword string
	NextcloudRoot     string
}

func Load() (Config, error) {
	cfg := Config{
		Environment:   valueOrDefault("APP_ENV", "development"),
		HTTPAddress:   valueOrDefault("HTTP_ADDRESS", ":8081"),
		DatabaseURL:   strings.TrimSpace(os.Getenv("DATABASE_URL")),
		OIDCIssuer:    strings.TrimRight(strings.TrimSpace(os.Getenv("OIDC_ISSUER")), "/"),
		OIDCJWKSURL:   strings.TrimSpace(os.Getenv("OIDC_JWKS_URL")),
		OIDCAudience:  strings.TrimSpace(os.Getenv("OIDC_API_AUDIENCE")),
		AllowedOrigin: valueOrDefault("CORS_ALLOWED_ORIGIN", "http://localhost:4200"),
		NextcloudURL: strings.TrimRight(
			strings.TrimSpace(os.Getenv("NEXTCLOUD_URL")),
			"/",
		),
		NextcloudUsername: strings.TrimSpace(os.Getenv("NEXTCLOUD_USERNAME")),
		NextcloudPassword: strings.TrimSpace(os.Getenv("NEXTCLOUD_PASSWORD")),
		NextcloudRoot:     valueOrDefault("NEXTCLOUD_ROOT", "DX-OS"),
	}

	var missing []string
	for name, value := range map[string]string{
		"DATABASE_URL":       cfg.DatabaseURL,
		"OIDC_ISSUER":        cfg.OIDCIssuer,
		"OIDC_JWKS_URL":      cfg.OIDCJWKSURL,
		"OIDC_API_AUDIENCE":  cfg.OIDCAudience,
		"NEXTCLOUD_URL":      cfg.NextcloudURL,
		"NEXTCLOUD_USERNAME": cfg.NextcloudUsername,
		"NEXTCLOUD_PASSWORD": cfg.NextcloudPassword,
	} {
		if value == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	if !strings.HasPrefix(cfg.OIDCIssuer, "http://") && !strings.HasPrefix(cfg.OIDCIssuer, "https://") {
		return Config{}, errors.New("OIDC_ISSUER must be an absolute HTTP(S) URL")
	}
	if !strings.HasPrefix(cfg.OIDCJWKSURL, "http://") && !strings.HasPrefix(cfg.OIDCJWKSURL, "https://") {
		return Config{}, errors.New("OIDC_JWKS_URL must be an absolute HTTP(S) URL")
	}
	if !strings.HasPrefix(cfg.NextcloudURL, "http://") && !strings.HasPrefix(cfg.NextcloudURL, "https://") {
		return Config{}, errors.New("NEXTCLOUD_URL must be an absolute HTTP(S) URL")
	}
	if strings.ContainsAny(cfg.NextcloudRoot, `/\`) || cfg.NextcloudRoot == "." || cfg.NextcloudRoot == ".." {
		return Config{}, errors.New("NEXTCLOUD_ROOT must be a single safe path segment")
	}

	return cfg, nil
}

func valueOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
