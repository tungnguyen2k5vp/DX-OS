package reporting

import (
	"errors"
	"testing"
	"time"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
)

func TestValidateDashboardInputNormalizesFilters(t *testing.T) {
	input := DashboardInput{
		From:       time.Date(2026, 7, 1, 15, 0, 0, 0, time.UTC),
		To:         time.Date(2026, 7, 31, 20, 0, 0, 0, time.UTC),
		CostCenter: " cc-general ",
		Currency:   "vnd",
	}
	if err := ValidateDashboardInput(&input); err != nil {
		t.Fatalf("expected valid filters, got %v", err)
	}
	if input.CostCenter != "CC-GENERAL" || input.Currency != "VND" {
		t.Fatalf("filters were not normalized: %#v", input)
	}
	if input.From.Hour() != 0 || input.To.Hour() != 0 {
		t.Fatalf("dates were not normalized: %#v", input)
	}
}

func TestValidateDashboardInputRejectsUnsafeRange(t *testing.T) {
	input := DashboardInput{
		From:         time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:           time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		DepartmentID: "not-a-uuid",
		CostCenter:   "CC GENERAL!",
		Currency:     "VN",
	}
	err := ValidateDashboardInput(&input)
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if len(validationError.Violations) != 4 {
		t.Fatalf("expected four violations, got %#v", validationError.Violations)
	}
}

func TestCanAccessReports(t *testing.T) {
	if !CanAccess(auth.Principal{Roles: []string{"finance"}}) {
		t.Fatal("finance should access reports")
	}
	if !CanAccess(auth.Principal{Roles: []string{"auditor"}}) {
		t.Fatal("auditor should access reports")
	}
	if CanAccess(auth.Principal{Roles: []string{"employee"}}) {
		t.Fatal("employee must not access reports")
	}
}
