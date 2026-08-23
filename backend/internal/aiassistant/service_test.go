package aiassistant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
)

func TestAskUsesRetrievedSourcesAndOllama(t *testing.T) {
	knowledgePath := t.TempDir()
	if err := os.WriteFile(filepath.Join(knowledgePath, "policy.md"), []byte("# Báo giá\nPhiếu mua sắm từ 20.000.000 VND phải có báo giá trước khi gửi duyệt."), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("unexpected Ollama path: %s", r.URL.Path)
		}
		var request struct {
			Messages []ollamaMessage `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.Messages) != 2 || request.Messages[1].Content == "" {
			t.Fatal("expected grounded Ollama messages")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"message": map[string]string{"role": "assistant", "content": "Cần báo giá [1]."}})
	}))
	defer server.Close()

	service := New(Config{Enabled: true, BaseURL: server.URL, ChatModel: "test-model", KnowledgePath: knowledgePath})
	answer, err := service.Ask(context.Background(), auth.Principal{Subject: "user-1", Roles: []string{"employee"}}, "Phiếu 20 triệu cần báo giá không?")
	if err != nil {
		t.Fatalf("Ask() unexpected error: %v", err)
	}
	if answer.Answer != "Cần báo giá [1]." || len(answer.Sources) != 1 || answer.Sources[0].Path != "policy.md" {
		t.Fatalf("unexpected grounded answer: %+v", answer)
	}
}

func TestAskRejectsQuestionWithoutEvidence(t *testing.T) {
	knowledgePath := t.TempDir()
	if err := os.WriteFile(filepath.Join(knowledgePath, "policy.md"), []byte("# Ngân sách\nNgân sách được giữ khi trưởng bộ phận phê duyệt."), 0o600); err != nil {
		t.Fatal(err)
	}
	service := New(Config{Enabled: true, BaseURL: "http://127.0.0.1:1", ChatModel: "test-model", KnowledgePath: knowledgePath})
	_, err := service.Ask(context.Background(), auth.Principal{Subject: "user-1", Roles: []string{"employee"}}, "thiên văn học lượng tử")
	if !errors.Is(err, ErrNoEvidence) {
		t.Fatalf("expected ErrNoEvidence, got %v", err)
	}
}

func TestAskRejectsOversizedQuestion(t *testing.T) {
	service := New(Config{Enabled: true})
	_, err := service.Ask(context.Background(), auth.Principal{Subject: "user-1", Roles: []string{"employee"}}, string(make([]rune, maxQuestionRunes+1)))
	if !errors.Is(err, ErrInvalidQuestion) {
		t.Fatalf("expected ErrInvalidQuestion, got %v", err)
	}
}
