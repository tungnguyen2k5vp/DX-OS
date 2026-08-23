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
	t.Setenv("AI_ENABLED", "true")
	t.Setenv("OLLAMA_URL", "http://host.docker.internal:11434/")
	t.Setenv("OLLAMA_CHAT_MODEL", "qwen3:4b-instruct")
	t.Setenv("AI_KNOWLEDGE_PATH", "/app/knowledge")
	t.Setenv("AI_REQUEST_TIMEOUT", "90s")

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
	if !cfg.AIEnabled || cfg.OllamaURL != "http://host.docker.internal:11434" || cfg.OllamaChatModel != "qwen3:4b-instruct" {
		t.Fatalf("unexpected local AI configuration: %+v", cfg)
	}
}

func TestLoadRejectsInvalidAIConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("OIDC_ISSUER", "http://localhost:8080/realms/dx-os")
	t.Setenv("OIDC_JWKS_URL", "http://keycloak:8080/certs")
	t.Setenv("OIDC_API_AUDIENCE", "dx-api")
	t.Setenv("NEXTCLOUD_URL", "http://nextcloud")
	t.Setenv("NEXTCLOUD_USERNAME", "dxos-nextcloud-admin")
	t.Setenv("NEXTCLOUD_PASSWORD", "secret")
	t.Setenv("AI_ENABLED", "true")
	t.Setenv("OLLAMA_URL", "not-a-url")

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error for invalid OLLAMA_URL")
	}
}
