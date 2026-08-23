package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dx-os-lab/dx-os/backend/internal/aiassistant"
	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
)

type assistantStub struct {
	question string
}

func (stub *assistantStub) Status(context.Context) aiassistant.Status {
	return aiassistant.Status{Enabled: true, Available: true, Provider: "ollama-local", Model: "test-model", KnowledgeDocuments: 2}
}

func (stub *assistantStub) Ask(_ context.Context, _ auth.Principal, question string) (aiassistant.Answer, error) {
	stub.question = question
	return aiassistant.Answer{Answer: "Cần báo giá [1].", Model: "test-model", GeneratedAt: time.Now(), Sources: []aiassistant.Source{{Index: 1, Path: "USER_GUIDE.md"}}}, nil
}

func TestAIAssistantRequiresAuthentication(t *testing.T) {
	handler := New(Dependencies{Assistant: &assistantStub{}, AllowedOrigin: "http://localhost:4200", Logger: slog.New(slog.NewTextHandler(discardWriter{}, nil)), TokenVerifier: verifierStub{}})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/ai/assistant/status", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestAIAssistantAnswersAuthenticatedQuestion(t *testing.T) {
	stub := &assistantStub{}
	handler := New(Dependencies{Assistant: stub, AllowedOrigin: "http://localhost:4200", Logger: slog.New(slog.NewTextHandler(discardWriter{}, nil)), TokenVerifier: verifierStub{}})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/assistant/questions", strings.NewReader(`{"question":"Phiếu 20 triệu cần gì?"}`))
	request.Header.Set("Authorization", "Bearer valid")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if stub.question != "Phiếu 20 triệu cần gì?" {
		t.Fatalf("unexpected question: %s", stub.question)
	}
}
