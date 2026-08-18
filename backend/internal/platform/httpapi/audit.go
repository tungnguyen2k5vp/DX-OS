package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dx-os-lab/dx-os/backend/internal/reporting"
)

func (a *api) getAuditEvents(w http.ResponseWriter, r *http.Request) {
	if a.reporting == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Dịch vụ chưa sẵn sàng", "Dịch vụ báo cáo hiện không sẵn sàng.")
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Cần access token hợp lệ để tiếp tục.")
		return
	}
	input := reporting.AuditInput{Page: 1, PageSize: 20}
	query := r.URL.Query()
	var violations []reporting.FieldViolation
	supported := map[string]bool{"page": true, "pageSize": true, "resourceType": true, "action": true, "from": true, "to": true}
	for key, values := range query {
		if !supported[key] {
			violations = append(violations, reporting.FieldViolation{Field: key, Message: "Tham số này không được hỗ trợ."})
		}
		if len(values) != 1 {
			violations = append(violations, reporting.FieldViolation{Field: key, Message: "Tham số chỉ được xuất hiện một lần."})
		}
	}
	if value := query.Get("page"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			violations = append(violations, reporting.FieldViolation{Field: "page", Message: "Phải là số nguyên."})
		} else {
			input.Page = parsed
		}
	}
	if value := query.Get("pageSize"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			violations = append(violations, reporting.FieldViolation{Field: "pageSize", Message: "Phải là số nguyên."})
		} else {
			input.PageSize = parsed
		}
	}
	input.ResourceType = query.Get("resourceType")
	input.Action = query.Get("action")
	if value := strings.TrimSpace(query.Get("from")); value != "" {
		parsed, err := time.Parse(time.DateOnly, value)
		if err != nil {
			violations = append(violations, reporting.FieldViolation{Field: "from", Message: "Phải có định dạng YYYY-MM-DD."})
		} else {
			input.From = &parsed
		}
	}
	if value := strings.TrimSpace(query.Get("to")); value != "" {
		parsed, err := time.Parse(time.DateOnly, value)
		if err != nil {
			violations = append(violations, reporting.FieldViolation{Field: "to", Message: "Phải có định dạng YYYY-MM-DD."})
		} else {
			endExclusive := parsed.AddDate(0, 0, 1)
			input.To = &endExclusive
		}
	}
	if err := reporting.ValidateAuditInput(&input); err != nil {
		var validationError *reporting.ValidationError
		if errors.As(err, &validationError) {
			violations = append(violations, validationError.Violations...)
		}
	}
	if len(violations) > 0 {
		writeValidationProblem(w, r, "invalid-audit-filters", "Một hoặc nhiều bộ lọc kiểm toán không hợp lệ.", violations)
		return
	}
	result, err := a.reporting.AuditCenter(r.Context(), principal, input)
	if err != nil {
		if errors.Is(err, reporting.ErrForbidden) {
			writeProblem(w, r, http.StatusForbidden, "audit-forbidden", "Không có quyền truy cập", "Tài khoản không có quyền xem bằng chứng kiểm toán.")
			return
		}
		a.logger.Error("audit center request failed", "error", err, "correlation_id", correlationIDFromContext(r.Context()))
		writeProblem(w, r, http.StatusInternalServerError, "internal", "Lỗi máy chủ", "Không thể tải bằng chứng kiểm toán.")
		return
	}
	writeJSON(w, http.StatusOK, result)
}
