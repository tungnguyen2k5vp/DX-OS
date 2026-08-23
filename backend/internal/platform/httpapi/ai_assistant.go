package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/dx-os-lab/dx-os/backend/internal/aiassistant"
)

type aiQuestionBody struct {
	Question string `json:"question"`
}

func (a *api) getAIAssistantStatus(w http.ResponseWriter, r *http.Request) {
	if a.assistant == nil {
		writeJSON(w, http.StatusOK, aiassistant.Status{Enabled: false, Provider: "ollama-local", Message: "Trợ lý AI local chưa được cấu hình."})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	writeJSON(w, http.StatusOK, a.assistant.Status(ctx))
}

func (a *api) askAIAssistant(w http.ResponseWriter, r *http.Request) {
	if a.assistant == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "ai-disabled", "Trợ lý AI chưa được bật", "Cấu hình AI local chưa sẵn sàng.")
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "missing-principal", "Chưa xác thực", "Không tìm thấy danh tính đã xác thực.")
		return
	}
	var body aiQuestionBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	body.Question = strings.TrimSpace(body.Question)
	startedAt := time.Now()
	result, err := a.assistant.Ask(r.Context(), principal, body.Question)
	if err != nil {
		switch {
		case errors.Is(err, aiassistant.ErrInvalidQuestion):
			writeValidationProblem(w, r, "invalid-ai-question", "Câu hỏi phải có từ 3 đến 1.000 ký tự.", nil)
		case errors.Is(err, aiassistant.ErrNoEvidence):
			writeProblem(w, r, http.StatusUnprocessableEntity, "ai-evidence-not-found", "Chưa đủ bằng chứng", "Không tìm thấy nội dung liên quan trong kho tài liệu nội bộ. Hãy đặt câu hỏi cụ thể hơn.")
		case errors.Is(err, aiassistant.ErrDisabled):
			writeProblem(w, r, http.StatusServiceUnavailable, "ai-disabled", "Trợ lý AI chưa được bật", "Bật AI_ENABLED và khởi động Ollama local trước khi sử dụng.")
		case errors.Is(err, aiassistant.ErrUnavailable):
			writeProblem(w, r, http.StatusServiceUnavailable, "ai-unavailable", "Mô hình AI local chưa sẵn sàng", "Không thể nhận câu trả lời từ Ollama. Kiểm tra tiến trình và model local rồi thử lại.")
		default:
			a.logger.Error("AI assistant request failed", "error", err, "subject", principal.Subject, "duration_ms", time.Since(startedAt).Milliseconds(), "correlation_id", correlationIDFromContext(r.Context()))
			writeProblem(w, r, http.StatusServiceUnavailable, "ai-unavailable", "Trợ lý AI chưa sẵn sàng", "Kho tri thức hoặc mô hình local đang tạm thời gián đoạn.")
		}
		return
	}
	a.logger.Info("AI assistant request completed", "subject", principal.Subject, "model", result.Model, "source_count", len(result.Sources), "duration_ms", result.DurationMS, "correlation_id", correlationIDFromContext(r.Context()))
	writeJSON(w, http.StatusOK, result)
}
