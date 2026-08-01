package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/dx-os-lab/dx-os/backend/internal/reporting"
)

type reportingStub struct {
	input     reporting.DashboardInput
	principal auth.Principal
	err       error
}

func (s *reportingStub) Dashboard(
	_ context.Context,
	principal auth.Principal,
	input reporting.DashboardInput,
) (reporting.Dashboard, error) {
	s.input = input
	s.principal = principal
	if s.err != nil {
		return reporting.Dashboard{}, s.err
	}
	return reporting.Dashboard{
		Filters: reporting.AppliedFilters{
			From: input.From.Format(time.DateOnly),
			To:   input.To.Format(time.DateOnly),
		},
		CurrencyTotals: []reporting.CurrencyTotal{},
		Statuses:       []reporting.StatusBreakdown{},
		Trends:         []reporting.DailyTrend{},
		Departments:    []reporting.DepartmentBreakdown{},
		Budgets:        []reporting.BudgetUtilization{},
	}, nil
}

func TestGetProcurementReportParsesFilters(t *testing.T) {
	service := &reportingStub{}
	handler := newReportingTestHandler(service)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/reports/procurement?from=2026-07-01&to=2026-07-31&costCenter=cc-general&currency=vnd",
		nil,
	)
	request.Header.Set("Authorization", "Bearer finance")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if service.input.CostCenter != "CC-GENERAL" ||
		service.input.Currency != "VND" ||
		service.input.From.Format(time.DateOnly) != "2026-07-01" {
		t.Fatalf("unexpected report filters: %#v", service.input)
	}
	if service.principal.Username != "finance.demo" {
		t.Fatalf("unexpected principal: %#v", service.principal)
	}
}

func TestGetProcurementReportRejectsInvalidFilters(t *testing.T) {
	handler := newReportingTestHandler(&reportingStub{})
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/reports/procurement?from=not-a-date&unknown=true",
		nil,
	)
	request.Header.Set("Authorization", "Bearer auditor")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", response.Code, response.Body.String())
	}
}

func TestGetProcurementReportMapsForbidden(t *testing.T) {
	handler := newReportingTestHandler(&reportingStub{err: reporting.ErrForbidden})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/reports/procurement", nil)
	request.Header.Set("Authorization", "Bearer finance")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", response.Code, response.Body.String())
	}
}

func newReportingTestHandler(service reportingService) http.Handler {
	return New(Dependencies{
		AllowedOrigin: "http://localhost:4200",
		Logger:        slog.New(slog.NewTextHandler(discardWriter{}, nil)),
		Reporting:     service,
		TokenVerifier: verifierStub{},
	})
}
