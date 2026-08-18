package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/dx-os-lab/dx-os/backend/internal/reporting"
)

func (a *api) getProcurementReport(w http.ResponseWriter, r *http.Request) {
	if a.reporting == nil {
		writeProblem(
			w, r, http.StatusServiceUnavailable, "service-unavailable",
			"Dịch vụ chưa sẵn sàng", "Dịch vụ báo cáo hiện không sẵn sàng.",
		)
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(
			w, r, http.StatusUnauthorized, "unauthenticated",
			"Cần đăng nhập", "Cần access token hợp lệ để tiếp tục.",
		)
		return
	}

	input := reporting.DefaultDashboardInput(time.Now())
	query := r.URL.Query()
	var violations []reporting.FieldViolation
	supported := map[string]bool{
		"from": true, "to": true, "departmentId": true, "costCenter": true, "currency": true,
	}
	for key, values := range query {
		if !supported[key] {
			violations = append(violations, reporting.FieldViolation{
				Field: key, Message: "Tham số không được hỗ trợ.",
			})
		}
		if len(values) != 1 {
			violations = append(violations, reporting.FieldViolation{
				Field: key, Message: "Tham số chỉ được xuất hiện một lần.",
			})
		}
	}
	if value := strings.TrimSpace(query.Get("from")); value != "" {
		parsed, err := time.Parse(time.DateOnly, value)
		if err != nil {
			violations = append(violations, reporting.FieldViolation{
				Field: "from", Message: "Ngày phải có định dạng YYYY-MM-DD.",
			})
		} else {
			input.From = parsed
		}
	}
	if value := strings.TrimSpace(query.Get("to")); value != "" {
		parsed, err := time.Parse(time.DateOnly, value)
		if err != nil {
			violations = append(violations, reporting.FieldViolation{
				Field: "to", Message: "Ngày phải có định dạng YYYY-MM-DD.",
			})
		} else {
			input.To = parsed
		}
	}
	input.DepartmentID = query.Get("departmentId")
	input.CostCenter = query.Get("costCenter")
	input.Currency = query.Get("currency")
	if err := reporting.ValidateDashboardInput(&input); err != nil {
		var validationError *reporting.ValidationError
		if errors.As(err, &validationError) {
			violations = append(violations, validationError.Violations...)
		}
	}
	if len(violations) > 0 {
		writeValidationProblem(
			w, r, "invalid-report-filters",
			"Một hoặc nhiều bộ lọc báo cáo không hợp lệ.",
			violations,
		)
		return
	}

	result, err := a.reporting.Dashboard(r.Context(), principal, input)
	if err != nil {
		switch {
		case errors.Is(err, reporting.ErrForbidden):
			writeProblem(
				w, r, http.StatusForbidden, "reporting-forbidden",
				"Không có quyền truy cập", "Tài khoản không có quyền xem báo cáo vận hành.",
			)
		default:
			a.logger.Error(
				"reporting request failed",
				"error", err,
				"correlation_id", correlationIDFromContext(r.Context()),
			)
			writeProblem(
				w, r, http.StatusInternalServerError, "internal",
				"Lỗi máy chủ", "Không thể tạo báo cáo vận hành.",
			)
		}
		return
	}
	writeJSON(w, http.StatusOK, result)
}
