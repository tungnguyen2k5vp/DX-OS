package httpapi

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/dx-os-lab/dx-os/backend/internal/procurement"
)

type procurementStub struct {
	createdInput      procurement.CreateInput
	listInput         procurement.ListInput
	timelineInput     procurement.TimelineInput
	budgetInput       procurement.BudgetSummaryInput
	updatedInput      procurement.UpdateInput
	transitionInput   procurement.TransitionInput
	adjustBudgetInput procurement.AdjustBudgetInput
	attachmentInput   procurement.UploadAttachmentInput
}

func (s *procurementStub) Create(
	_ context.Context,
	_ auth.Principal,
	input procurement.CreateInput,
	_ string,
) (procurement.PurchaseRequest, error) {
	s.createdInput = input
	return procurement.PurchaseRequest{
		ID:          "f24e668b-f62a-4f5f-88c9-2fb3c3973106",
		RequestCode: "PR-2026-000001",
		Status:      procurement.StatusDraft,
	}, nil
}

func (s *procurementStub) List(
	_ context.Context,
	_ auth.Principal,
	input procurement.ListInput,
) (procurement.ListResult, error) {
	s.listInput = input
	return procurement.ListResult{
		Items:    []procurement.PurchaseRequest{},
		Page:     input.Page,
		PageSize: input.PageSize,
	}, nil
}

func (s *procurementStub) Get(
	_ context.Context,
	_ auth.Principal,
	requestID string,
) (procurement.PurchaseRequest, error) {
	return procurement.PurchaseRequest{ID: requestID, Status: procurement.StatusDraft}, nil
}

func (s *procurementStub) Timeline(
	_ context.Context,
	_ auth.Principal,
	_ string,
	input procurement.TimelineInput,
) (procurement.TimelineResult, error) {
	s.timelineInput = input
	return procurement.TimelineResult{
		Items:    []procurement.TimelineEvent{},
		Page:     input.Page,
		PageSize: input.PageSize,
	}, nil
}

func (s *procurementStub) GetBudgetSummary(
	_ context.Context,
	_ auth.Principal,
	input procurement.BudgetSummaryInput,
) (procurement.BudgetSummary, error) {
	s.budgetInput = input
	return procurement.BudgetSummary{
		CostCenter: input.CostCenter,
		Currency:   input.Currency,
	}, nil
}

func (s *procurementStub) BudgetCheck(
	_ context.Context,
	_ auth.Principal,
	_ string,
) (procurement.BudgetCheck, error) {
	return procurement.BudgetCheck{Configured: true, Result: "AVAILABLE"}, nil
}

func (s *procurementStub) BudgetDashboard(
	_ context.Context,
	_ auth.Principal,
) (procurement.BudgetDashboard, error) {
	return procurement.BudgetDashboard{
		Allocations:  []procurement.BudgetAllocation{},
		Totals:       []procurement.BudgetCurrencyTotal{},
		Reservations: []procurement.BudgetReservation{},
		Adjustments:  []procurement.BudgetAdjustment{},
		CanManage:    true,
	}, nil
}

func (s *procurementStub) AdjustBudget(
	_ context.Context,
	_ auth.Principal,
	allocationID string,
	input procurement.AdjustBudgetInput,
) (procurement.BudgetAllocation, error) {
	s.adjustBudgetInput = input
	return procurement.BudgetAllocation{
		ID: allocationID, AllocatedAmount: input.AllocatedAmount,
	}, nil
}

func (s *procurementStub) Update(
	_ context.Context,
	_ auth.Principal,
	requestID string,
	input procurement.UpdateInput,
	_ string,
) (procurement.PurchaseRequest, error) {
	s.updatedInput = input
	return procurement.PurchaseRequest{
		ID: requestID, Status: procurement.StatusDraft, Version: input.ExpectedVersion + 1,
	}, nil
}

func (s *procurementStub) Transition(
	_ context.Context,
	_ auth.Principal,
	requestID string,
	input procurement.TransitionInput,
) (procurement.PurchaseRequest, error) {
	s.transitionInput = input
	return procurement.PurchaseRequest{
		ID: requestID, Status: procurement.StatusSubmitted, Version: input.ExpectedVersion + 1,
	}, nil
}

func (s *procurementStub) UploadAttachment(
	_ context.Context,
	_ auth.Principal,
	requestID string,
	input procurement.UploadAttachmentInput,
) (procurement.Attachment, error) {
	s.attachmentInput = input
	return procurement.Attachment{
		ID:           "d12c5b6f-0e99-4da8-83d7-ec86dd36a044",
		PurchaseID:   requestID,
		DocumentType: input.DocumentType,
		FileName:     input.FileName,
		ContentType:  input.ContentType,
		SizeBytes:    int64(len(input.Content)),
	}, nil
}

func (s *procurementStub) ListAttachments(
	_ context.Context,
	_ auth.Principal,
	_ string,
) (procurement.AttachmentList, error) {
	return procurement.AttachmentList{
		Items:               []procurement.Attachment{},
		MaxSizeBytes:        procurement.MaxAttachmentSize,
		AllowedContentTypes: procurement.AllowedAttachmentContentTypes,
	}, nil
}

func (s *procurementStub) DownloadAttachment(
	_ context.Context,
	_ auth.Principal,
	requestID string,
	attachmentID string,
) (procurement.AttachmentContent, error) {
	return procurement.AttachmentContent{
		Attachment: procurement.Attachment{
			ID:          attachmentID,
			PurchaseID:  requestID,
			FileName:    "bao-gia.pdf",
			ContentType: "application/pdf",
		},
		Content: []byte("%PDF-test"),
	}, nil
}

func (s *procurementStub) DeleteAttachment(
	_ context.Context,
	_ auth.Principal,
	_ string,
	_ string,
	_ string,
) error {
	return nil
}

func TestCreatePurchaseRequest(t *testing.T) {
	service := &procurementStub{}
	handler := newProcurementTestHandler(service)
	body := []byte(`{
		"title":"Laptop for design team",
		"reason":"Required for the approved design workload.",
		"currency":"VND",
		"costCenter":"CC-DESIGN",
		"items":[
			{
				"description":"Developer laptop",
				"quantity":"2",
				"unit":"unit",
				"unitPrice":"25000000"
			}
		]
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/purchase-requests", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer valid")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	if service.createdInput.Currency != "VND" || len(service.createdInput.Items) != 1 {
		t.Fatalf("unexpected create input: %#v", service.createdInput)
	}
	if response.Header().Get("Location") == "" {
		t.Fatal("expected Location response header")
	}
}

func TestCreatePurchaseRequestRejectsUnknownFields(t *testing.T) {
	handler := newProcurementTestHandler(&procurementStub{})
	body := []byte(`{"title":"Valid title","unexpected":true}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/purchase-requests", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer valid")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
}

func TestListPurchaseRequestsParsesPaginationAndStatus(t *testing.T) {
	service := &procurementStub{}
	handler := newProcurementTestHandler(service)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/purchase-requests?page=2&pageSize=10&status=draft",
		nil,
	)
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if service.listInput.Page != 2 || service.listInput.PageSize != 10 {
		t.Fatalf("unexpected pagination: %#v", service.listInput)
	}
	if service.listInput.Status == nil || *service.listInput.Status != procurement.StatusDraft {
		t.Fatalf("unexpected status: %#v", service.listInput.Status)
	}
}

func TestGetPurchaseRequestRejectsInvalidUUID(t *testing.T) {
	handler := newProcurementTestHandler(&procurementStub{})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/purchase-requests/not-a-uuid", nil)
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", response.Code)
	}
}

func TestGetPurchaseRequestTimelineParsesPagination(t *testing.T) {
	service := &procurementStub{}
	handler := newProcurementTestHandler(service)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/purchase-requests/f24e668b-f62a-4f5f-88c9-2fb3c3973106/timeline?page=2&pageSize=10",
		nil,
	)
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if service.timelineInput.Page != 2 || service.timelineInput.PageSize != 10 {
		t.Fatalf("unexpected timeline pagination: %#v", service.timelineInput)
	}
}

func TestGetPurchaseRequestTimelineRejectsUnknownQuery(t *testing.T) {
	handler := newProcurementTestHandler(&procurementStub{})
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/purchase-requests/f24e668b-f62a-4f5f-88c9-2fb3c3973106/timeline?includeMetadata=true",
		nil,
	)
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", response.Code, response.Body.String())
	}
}

func TestGetPurchaseRequestBudgetCheck(t *testing.T) {
	handler := newProcurementTestHandler(&procurementStub{})
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/purchase-requests/f24e668b-f62a-4f5f-88c9-2fb3c3973106/budget-check",
		nil,
	)
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
}

func TestGetBudgetSummaryValidatesAndNormalizesQuery(t *testing.T) {
	service := &procurementStub{}
	handler := newProcurementTestHandler(service)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/budgets/summary?costCenter=CC-GENERAL&currency=vnd",
		nil,
	)
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if service.budgetInput.CostCenter != "CC-GENERAL" || service.budgetInput.Currency != "VND" {
		t.Fatalf("unexpected budget query: %#v", service.budgetInput)
	}
}

func TestGetBudgetDashboard(t *testing.T) {
	handler := newProcurementTestHandler(&procurementStub{})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/budgets/dashboard", nil)
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
}

func TestAdjustBudgetPassesVersionAndIdempotency(t *testing.T) {
	service := &procurementStub{}
	handler := newProcurementTestHandler(service)
	body := []byte(`{
		"allocatedAmount":"120000000000",
		"expectedVersion":3,
		"reason":"Approved annual budget increase."
	}`)
	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/budgets/allocations/00000000-0000-4000-8000-000000000301",
		bytes.NewReader(body),
	)
	request.Header.Set("Authorization", "Bearer valid")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "budget-adjustment-0001")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if service.adjustBudgetInput.ExpectedVersion != 3 ||
		service.adjustBudgetInput.IdempotencyKey != "budget-adjustment-0001" {
		t.Fatalf("unexpected adjustment input: %#v", service.adjustBudgetInput)
	}
}

func TestUpdatePurchaseRequestPassesExpectedVersion(t *testing.T) {
	service := &procurementStub{}
	handler := newProcurementTestHandler(service)
	body := []byte(`{
		"title":"Updated laptop request",
		"reason":"Updated business justification for the design workload.",
		"currency":"VND",
		"costCenter":"CC-DESIGN",
		"expectedVersion":3,
		"items":[{
			"description":"Developer laptop",
			"quantity":"2",
			"unit":"unit",
			"unitPrice":"25000000"
		}]
	}`)
	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/purchase-requests/f24e668b-f62a-4f5f-88c9-2fb3c3973106",
		bytes.NewReader(body),
	)
	request.Header.Set("Authorization", "Bearer valid")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if service.updatedInput.ExpectedVersion != 3 {
		t.Fatalf("unexpected update input: %#v", service.updatedInput)
	}
}

func TestTransitionPurchaseRequestRequiresIdempotencyKey(t *testing.T) {
	handler := newProcurementTestHandler(&procurementStub{})
	body := []byte(`{"action":"SUBMIT","expectedVersion":1,"comment":""}`)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/purchase-requests/f24e668b-f62a-4f5f-88c9-2fb3c3973106/transitions",
		bytes.NewReader(body),
	)
	request.Header.Set("Authorization", "Bearer valid")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", response.Code, response.Body.String())
	}
}

func TestTransitionPurchaseRequestPassesIdempotencyKey(t *testing.T) {
	service := &procurementStub{}
	handler := newProcurementTestHandler(service)
	body := []byte(`{"action":"SUBMIT","expectedVersion":1,"comment":""}`)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/purchase-requests/f24e668b-f62a-4f5f-88c9-2fb3c3973106/transitions",
		bytes.NewReader(body),
	)
	request.Header.Set("Authorization", "Bearer valid")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "submit-test-0001")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if service.transitionInput.IdempotencyKey != "submit-test-0001" ||
		service.transitionInput.Action != procurement.ActionSubmit {
		t.Fatalf("unexpected transition input: %#v", service.transitionInput)
	}
}

func newProcurementTestHandler(service purchaseRequestService) http.Handler {
	return New(Dependencies{
		AllowedOrigin: "http://localhost:4200",
		Logger:        slog.New(slog.NewTextHandler(discardWriter{}, nil)),
		Procurement:   service,
		TokenVerifier: verifierStub{},
	})
}
