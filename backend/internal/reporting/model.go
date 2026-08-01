package reporting

import (
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
)

var (
	ErrForbidden = errors.New("reporting access is forbidden")
	uuidPattern  = regexp.MustCompile(
		`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`,
	)
	costCenterPattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9._-]{1,99}$`)
	currencyPattern   = regexp.MustCompile(`^[A-Z]{3}$`)
)

type FieldViolation struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationError struct {
	Violations []FieldViolation
}

func (e *ValidationError) Error() string {
	return "report filter validation failed"
}

type DashboardInput struct {
	From         time.Time
	To           time.Time
	DepartmentID string
	CostCenter   string
	Currency     string
}

type AppliedFilters struct {
	From         string `json:"from"`
	To           string `json:"to"`
	DepartmentID string `json:"departmentId,omitempty"`
	CostCenter   string `json:"costCenter,omitempty"`
	Currency     string `json:"currency,omitempty"`
}

type Summary struct {
	TotalRequests            int64  `json:"totalRequests"`
	ApprovedCount            int64  `json:"approvedCount"`
	RejectedCount            int64  `json:"rejectedCount"`
	ReturnedCount            int64  `json:"returnedCount"`
	SLABreachedCount         int64  `json:"slaBreachedCount"`
	AverageLeadTimeHours     string `json:"averageLeadTimeHours"`
	AttachmentRequiredCount  int64  `json:"attachmentRequiredCount"`
	AttachmentCompliantCount int64  `json:"attachmentCompliantCount"`
	AttachmentComplianceRate string `json:"attachmentComplianceRate"`
}

type CurrencyTotal struct {
	Currency     string `json:"currency"`
	RequestCount int64  `json:"requestCount"`
	TotalAmount  string `json:"totalAmount"`
}

type StatusBreakdown struct {
	Status       string `json:"status"`
	Currency     string `json:"currency"`
	RequestCount int64  `json:"requestCount"`
	TotalAmount  string `json:"totalAmount"`
}

type DailyTrend struct {
	Date          string `json:"date"`
	Currency      string `json:"currency"`
	RequestCount  int64  `json:"requestCount"`
	ApprovedCount int64  `json:"approvedCount"`
	TotalAmount   string `json:"totalAmount"`
}

type DepartmentBreakdown struct {
	DepartmentID   string `json:"departmentId"`
	DepartmentName string `json:"departmentName"`
	Currency       string `json:"currency"`
	RequestCount   int64  `json:"requestCount"`
	ApprovedCount  int64  `json:"approvedCount"`
	TotalAmount    string `json:"totalAmount"`
}

type BudgetUtilization struct {
	PeriodCode         string `json:"periodCode"`
	PeriodStart        string `json:"periodStart"`
	PeriodEnd          string `json:"periodEnd"`
	CostCenter         string `json:"costCenter"`
	Currency           string `json:"currency"`
	AllocatedAmount    string `json:"allocatedAmount"`
	ReservedAmount     string `json:"reservedAmount"`
	CommittedAmount    string `json:"committedAmount"`
	AvailableAmount    string `json:"availableAmount"`
	UtilizationPercent string `json:"utilizationPercent"`
}

type Dashboard struct {
	Filters        AppliedFilters        `json:"filters"`
	Summary        Summary               `json:"summary"`
	CurrencyTotals []CurrencyTotal       `json:"currencyTotals"`
	Statuses       []StatusBreakdown     `json:"statuses"`
	Trends         []DailyTrend          `json:"trends"`
	Departments    []DepartmentBreakdown `json:"departments"`
	Budgets        []BudgetUtilization   `json:"budgets"`
	GeneratedAt    time.Time             `json:"generatedAt"`
}

func DefaultDashboardInput(now time.Time) DashboardInput {
	to := dateOnly(now.UTC())
	return DashboardInput{
		From: to.AddDate(0, 0, -29),
		To:   to,
	}
}

func ValidateDashboardInput(input *DashboardInput) error {
	var violations []FieldViolation
	if input.From.IsZero() {
		violations = append(violations, FieldViolation{Field: "from", Message: "Ngày bắt đầu là bắt buộc."})
	}
	if input.To.IsZero() {
		violations = append(violations, FieldViolation{Field: "to", Message: "Ngày kết thúc là bắt buộc."})
	}
	if !input.From.IsZero() && !input.To.IsZero() {
		input.From = dateOnly(input.From)
		input.To = dateOnly(input.To)
		if input.To.Before(input.From) {
			violations = append(violations, FieldViolation{
				Field: "to", Message: "Ngày kết thúc không được trước ngày bắt đầu.",
			})
		} else if input.To.Sub(input.From) > 366*24*time.Hour {
			violations = append(violations, FieldViolation{
				Field: "to", Message: "Khoảng báo cáo không được vượt quá 367 ngày.",
			})
		}
	}
	input.DepartmentID = strings.TrimSpace(input.DepartmentID)
	if input.DepartmentID != "" && !uuidPattern.MatchString(input.DepartmentID) {
		violations = append(violations, FieldViolation{
			Field: "departmentId", Message: "Phòng ban phải là UUID hợp lệ.",
		})
	}
	input.CostCenter = strings.ToUpper(strings.TrimSpace(input.CostCenter))
	if input.CostCenter != "" && !costCenterPattern.MatchString(input.CostCenter) {
		violations = append(violations, FieldViolation{
			Field: "costCenter", Message: "Cost center không hợp lệ.",
		})
	}
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	if input.Currency != "" && !currencyPattern.MatchString(input.Currency) {
		violations = append(violations, FieldViolation{
			Field: "currency", Message: "Tiền tệ phải có đúng 3 chữ cái.",
		})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func CanAccess(principal auth.Principal) bool {
	return slices.Contains(principal.Roles, "finance") ||
		slices.Contains(principal.Roles, "auditor") ||
		slices.Contains(principal.Roles, "dx_admin")
}

func isFinanceOnly(principal auth.Principal) bool {
	return slices.Contains(principal.Roles, "finance") &&
		!slices.Contains(principal.Roles, "auditor") &&
		!slices.Contains(principal.Roles, "dx_admin")
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
