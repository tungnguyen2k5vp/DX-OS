package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
)

type verifierStub struct{}

func (verifierStub) Verify(_ context.Context, rawToken string) (auth.Principal, error) {
	switch rawToken {
	case "valid":
		return auth.Principal{Subject: "123", Username: "employee.demo", Roles: []string{"employee"}}, nil
	case "finance":
		return auth.Principal{Subject: "456", Username: "finance.demo", Roles: []string{"finance"}}, nil
	case "auditor":
		return auth.Principal{Subject: "789", Username: "auditor.demo", Roles: []string{"auditor"}}, nil
	}
	return auth.Principal{}, auth.ErrInvalidToken
}

func TestMeRequiresBearerToken(t *testing.T) {
	handler := New(Dependencies{
		AllowedOrigin: "http://localhost:4200",
		Logger:        slog.New(slog.NewTextHandler(discardWriter{}, nil)),
		TokenVerifier: verifierStub{},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
	if response.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("unexpected content type: %s", response.Header().Get("Content-Type"))
	}
}

func TestMeReturnsAuthenticatedPrincipal(t *testing.T) {
	handler := New(Dependencies{
		AllowedOrigin: "http://localhost:4200",
		Logger:        slog.New(slog.NewTextHandler(discardWriter{}, nil)),
		TokenVerifier: verifierStub{},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
}

type discardWriter struct{}

func (discardWriter) Write(value []byte) (int, error) {
	return len(value), nil
}
