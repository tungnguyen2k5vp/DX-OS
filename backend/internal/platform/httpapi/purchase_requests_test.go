package httpapi

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/dx-os-lab/dx-os/backend/internal/procurement"
)

type procurementStub struct {
	createdInput      procurement.CreateInput
	listInput         procurement.ListInput
	timelineInput     procurement.TimelineInput
	commentInput      procurement.CommentInput
	budgetInput       procurement.BudgetSummaryInput
	updatedInput      procurement.UpdateInput
	transitionInput   procurement.TransitionInput
	adjustBudgetInput procurement.AdjustBudgetInput
	attachmentInput   procurement.UploadAttachmentInput
	supplierInput     procurement.SupplierInput
	orderInput        procurement.CreatePurchaseOrderInput
	receiptInput      procurement.ConfirmReceiptInput
	invoiceInput      procurement.InvoiceInput
	invoiceUpdate     procurement.UpdateInvoiceInput
	invoiceAction     procurement.InvoiceActionInput
	slaPolicyInput    procurement.UpdateSLAPolicyInput
	attachmentPolicy  procurement.UpdateAttachmentPolicyInput
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

func (s *procurementStub) ListComments(
	_ context.Context,
	_ auth.Principal,
	_ string,
) (procurement.CommentList, error) {
	return procurement.CommentList{Items: []procurement.Comment{}}, nil
}

func (s *procurementStub) AddComment(
	_ context.Context,
	_ auth.Principal,
	_ string,
	input procurement.CommentInput,
) (procurement.Comment, error) {
	s.commentInput = input
	return procurement.Comment{ID: "2f2237c0-4232-41a7-b55e-faf86f1a7f30", Body: input.Body}, nil
}

func (s *procurementStub) TaskSummary(
	_ context.Context,
	_ auth.Principal,
) (procurement.WorkSummary, error) {
	return procurement.WorkSummary{Items: []procurement.WorkTask{}}, nil
}

func (s *procurementStub) ListSuppliers(
	_ context.Context,
	_ auth.Principal,
) (procurement.SupplierList, error) {
	return procurement.SupplierList{Items: []procurement.Supplier{}}, nil
}

func (s *procurementStub) CreateSupplier(
	_ context.Context,
	_ auth.Principal,
	input procurement.SupplierInput,
	_ string,
) (procurement.Supplier, error) {
	s.supplierInput = input
	return procurement.Supplier{ID: "00000000-0000-4000-8000-000000000801", Code: input.Code}, nil
}

func (s *procurementStub) UpdateSupplier(
	_ context.Context,
	_ auth.Principal,
	supplierID string,
	input procurement.UpdateSupplierInput,
	_ string,
) (procurement.Supplier, error) {
	return procurement.Supplier{ID: supplierID, Code: input.Code, Version: input.ExpectedVersion + 1}, nil
}

func (s *procurementStub) OperationsBoard(
	_ context.Context,
	_ auth.Principal,
) (procurement.OperationsBoard, error) {
	return procurement.OperationsBoard{Items: []procurement.PurchaseOrder{}}, nil
}

func (s *procurementStub) CreatePurchaseOrder(
	_ context.Context,
	_ auth.Principal,
	input procurement.CreatePurchaseOrderInput,
) (procurement.PurchaseOrder, error) {
	s.orderInput = input
	return procurement.PurchaseOrder{PurchaseRequestID: input.PurchaseRequestID, Status: "ORDERED"}, nil
}

func (s *procurementStub) ConfirmReceipt(
	_ context.Context,
	_ auth.Principal,
	requestID string,
	input procurement.ConfirmReceiptInput,
) (procurement.PurchaseOrder, error) {
	s.receiptInput = input
	return procurement.PurchaseOrder{PurchaseRequestID: requestID, Status: "RECEIVED"}, nil
}

func (s *procurementStub) InvoiceBoard(
	_ context.Context,
	_ auth.Principal,
) (procurement.InvoiceBoard, error) {
	return procurement.InvoiceBoard{Items: []procurement.InvoiceBoardItem{}}, nil
}

func (s *procurementStub) CreateInvoice(
	_ context.Context,
	_ auth.Principal,
	input procurement.InvoiceInput,
) (procurement.InvoiceBoardItem, error) {
	s.invoiceInput = input
	id := "00000000-0000-4000-8000-000000001001"
	status := "RECORDED"
	return procurement.InvoiceBoardItem{InvoiceID: &id, InvoiceStatus: &status}, nil
}

func (s *procurementStub) UpdateInvoice(
	_ context.Context,
	_ auth.Principal,
	invoiceID string,
	input procurement.UpdateInvoiceInput,
) (procurement.InvoiceBoardItem, error) {
	s.invoiceUpdate = input
	return procurement.InvoiceBoardItem{InvoiceID: &invoiceID, Version: input.ExpectedVersion + 1}, nil
}

func (s *procurementStub) TransitionInvoice(
	_ context.Context,
	_ auth.Principal,
	invoiceID string,
	input procurement.InvoiceActionInput,
) (procurement.InvoiceBoardItem, error) {
	s.invoiceAction = input
	status := "VERIFIED"
	return procurement.InvoiceBoardItem{InvoiceID: &invoiceID, InvoiceStatus: &status}, nil
}

func (s *procurementStub) PolicyCenter(_ context.Context, _ auth.Principal) (procurement.PolicyCenter, error) {
	return procurement.PolicyCenter{
		SLA:             []procurement.SLAPolicy{{ProcessName: "PURCHASE_REQUEST_APPROVAL", TargetHours: 72, Active: true, Version: 1}},
		AttachmentRules: []procurement.AttachmentPolicy{}, CanManage: true,
	}, nil
}

func (s *procurementStub) UpdateSLAPolicy(_ context.Context, _ auth.Principal, processName string, input procurement.UpdateSLAPolicyInput) (procurement.SLAPolicy, error) {
	s.slaPolicyInput = input
	return procurement.SLAPolicy{ProcessName: processName, TargetHours: input.TargetHours, Active: input.Active, Version: input.ExpectedVersion + 1}, nil
}

func (s *procurementStub) UpdateAttachmentPolicy(_ context.Context, _ auth.Principal, ruleID string, input procurement.UpdateAttachmentPolicyInput) (procurement.AttachmentPolicy, error) {
	s.attachmentPolicy = input
	return procurement.AttachmentPolicy{ID: ruleID, ThresholdAmount: input.ThresholdAmount, RequiredDocumentType: input.RequiredDocumentType, Active: input.Active, Version: input.ExpectedVersion + 1}, nil
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

func TestParseAttachmentUploadStreamsBoundedMultipart(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("documentType", "QUOTATION"); err != nil {
		t.Fatal(err)
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="quotation.pdf"`)
	header.Set("Content-Type", "application/pdf")
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write([]byte("%PDF-test")); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	upload, err := parseAttachmentUpload(request)
	if err != nil {
		t.Fatalf("expected valid multipart upload, got %v", err)
	}
	if upload.documentType != "QUOTATION" || upload.fileName != "quotation.pdf" ||
		upload.contentType != "application/pdf" || string(upload.content) != "%PDF-test" {
		t.Fatalf("unexpected upload: %#v", upload)
	}
}

func TestParseAttachmentUploadRequiresFile(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("documentType", "QUOTATION"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	_, err := parseAttachmentUpload(request)
	if !errors.Is(err, errAttachmentFileRequired) {
		t.Fatalf("expected missing file error, got %v", err)
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

func TestAddPurchaseRequestComment(t *testing.T) {
	service := &procurementStub{}
	handler := newProcurementTestHandler(service)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/purchase-requests/f24e668b-f62a-4f5f-88c9-2fb3c3973106/comments",
		bytes.NewReader([]byte(`{"body":"  Please confirm the delivery date.  "}`)),
	)
	request.Header.Set("Authorization", "Bearer valid")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	if service.commentInput.Body != "  Please confirm the delivery date.  " {
		t.Fatalf("unexpected comment input: %#v", service.commentInput)
	}
}

func TestGetTaskSummary(t *testing.T) {
	handler := newProcurementTestHandler(&procurementStub{})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/tasks-summary", nil)
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
}

func TestCreateSupplierAndPurchaseOrderContracts(t *testing.T) {
	service := &procurementStub{}
	handler := newProcurementTestHandler(service)
	supplierRequest := httptest.NewRequest(
		http.MethodPost, "/api/v1/suppliers",
		bytes.NewReader([]byte(`{"code":"VEN-01","name":"Demo Vendor","taxCode":"TAX-01","contactName":"Sales","email":"sales@example.com","phone":"0900000000","status":"ACTIVE","riskLevel":"LOW"}`)),
	)
	supplierRequest.Header.Set("Authorization", "Bearer finance")
	supplierRequest.Header.Set("Content-Type", "application/json")
	supplierResponse := httptest.NewRecorder()
	handler.ServeHTTP(supplierResponse, supplierRequest)
	if supplierResponse.Code != http.StatusCreated || service.supplierInput.Code != "VEN-01" {
		t.Fatalf("unexpected supplier response %d: %s", supplierResponse.Code, supplierResponse.Body.String())
	}

	orderRequest := httptest.NewRequest(
		http.MethodPost, "/api/v1/procurement-operations/orders",
		bytes.NewReader([]byte(`{"purchaseRequestId":"f24e668b-f62a-4f5f-88c9-2fb3c3973106","supplierId":"00000000-0000-4000-8000-000000000801","externalReference":"ERP-01","expectedDeliveryOn":"2026-12-30","note":"Deliver to reception."}`)),
	)
	orderRequest.Header.Set("Authorization", "Bearer finance")
	orderRequest.Header.Set("Content-Type", "application/json")
	orderRequest.Header.Set("Idempotency-Key", "purchase-order-0001")
	orderResponse := httptest.NewRecorder()
	handler.ServeHTTP(orderResponse, orderRequest)
	if orderResponse.Code != http.StatusCreated || service.orderInput.IdempotencyKey != "purchase-order-0001" {
		t.Fatalf("unexpected order response %d: %s", orderResponse.Code, orderResponse.Body.String())
	}
}

func TestConfirmReceiptContract(t *testing.T) {
	service := &procurementStub{}
	handler := newProcurementTestHandler(service)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/procurement-operations/orders/f24e668b-f62a-4f5f-88c9-2fb3c3973106/receipt",
		bytes.NewReader([]byte(`{"expectedVersion":2,"actualDeliveryOn":"2026-08-14"}`)),
	)
	request.Header.Set("Authorization", "Bearer valid")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || service.receiptInput.ExpectedVersion != 2 {
		t.Fatalf("unexpected receipt response %d: %s", response.Code, response.Body.String())
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
