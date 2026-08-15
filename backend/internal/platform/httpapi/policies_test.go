package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPolicyCenterContract(t *testing.T) {
	service := &procurementStub{}
	handler := newProcurementTestHandler(service)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/policies", nil)
	request.Header.Set("Authorization", "Bearer auditor")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
}

func TestUpdateSLAPolicyContract(t *testing.T) {
	service := &procurementStub{}
	handler := newProcurementTestHandler(service)
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/policies/sla/PURCHASE_REQUEST_APPROVAL", bytes.NewBufferString(`{"targetHours":48,"active":true,"expectedVersion":2}`))
	request.Header.Set("Authorization", "Bearer valid")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if service.slaPolicyInput.TargetHours != 48 || service.slaPolicyInput.ExpectedVersion != 2 {
		t.Fatalf("unexpected SLA input: %#v", service.slaPolicyInput)
	}
}

func TestUpdateAttachmentPolicyContract(t *testing.T) {
	service := &procurementStub{}
	handler := newProcurementTestHandler(service)
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/policies/attachments/00000000-0000-4000-8000-000000001101", bytes.NewBufferString(`{"thresholdAmount":"30000000","requiredDocumentType":"QUOTATION","active":true,"expectedVersion":1}`))
	request.Header.Set("Authorization", "Bearer valid")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if service.attachmentPolicy.ThresholdAmount != "30000000" {
		t.Fatalf("unexpected attachment policy input: %#v", service.attachmentPolicy)
	}
}
