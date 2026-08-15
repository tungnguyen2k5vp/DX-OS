package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInvoiceHTTPContracts(t *testing.T) {
	service := &procurementStub{}
	handler := newProcurementTestHandler(service)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil)
	request.Header.Set("Authorization", "Bearer finance")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("invoice board expected 200, got %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString(`{
		"purchaseOrderId":"00000000-0000-4000-8000-000000000802",
		"invoiceNumber":"INV-2026-001","issuedOn":"2026-08-15","dueOn":"2026-09-15",
		"amount":"27000000","currency":"VND","note":"Demo invoice"
	}`))
	request.Header.Set("Authorization", "Bearer finance")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "invoice-create-0001")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create invoice expected 201, got %d: %s", response.Code, response.Body.String())
	}
	if service.invoiceInput.InvoiceNumber != "INV-2026-001" || service.invoiceInput.IdempotencyKey != "invoice-create-0001" {
		t.Fatalf("unexpected invoice input: %+v", service.invoiceInput)
	}

	invoiceID := "00000000-0000-4000-8000-000000001001"
	request = httptest.NewRequest(http.MethodPost, "/api/v1/invoices/"+invoiceID+"/transitions", bytes.NewBufferString(`{
		"action":"MARK_PAID","expectedVersion":2,"comment":"Paid",
		"paymentReference":"BANK-001","paidOn":"2026-08-15"
	}`))
	request.Header.Set("Authorization", "Bearer finance")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "invoice-payment-0001")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("invoice transition expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if service.invoiceAction.Action != "MARK_PAID" || service.invoiceAction.PaymentReference != "BANK-001" {
		t.Fatalf("unexpected invoice action: %+v", service.invoiceAction)
	}
}
