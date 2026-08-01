package config

import "testing"

func TestLoadRequiresSecurityConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("OIDC_ISSUER", "")
	t.Setenv("OIDC_JWKS_URL", "")
	t.Setenv("OIDC_API_AUDIENCE", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error for missing required configuration")
	}
}

func TestLoadAcceptsValidConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("OIDC_ISSUER", "http://localhost:8080/realms/dx-os/")
	t.Setenv("OIDC_JWKS_URL", "http://keycloak:8080/certs")
	t.Setenv("OIDC_API_AUDIENCE", "dx-api")
	t.Setenv("NEXTCLOUD_URL", "http://nextcloud")
	t.Setenv("NEXTCLOUD_USERNAME", "dxos-nextcloud-admin")
	t.Setenv("NEXTCLOUD_PASSWORD", "secret")
	t.Setenv("NEXTCLOUD_ROOT", "DX-OS")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.OIDCIssuer != "http://localhost:8080/realms/dx-os" {
		t.Fatalf("unexpected issuer: %s", cfg.OIDCIssuer)
	}
	if cfg.NextcloudURL != "http://nextcloud" {
		t.Fatalf("unexpected Nextcloud URL: %s", cfg.NextcloudURL)
	}
}
