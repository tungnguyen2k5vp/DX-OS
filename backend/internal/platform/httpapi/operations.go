package httpapi

import (
	"net/http"
	"strings"

	"github.com/dx-os-lab/dx-os/backend/internal/procurement"
	"github.com/go-chi/chi/v5"
)

type supplierBody struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	TaxCode         string `json:"taxCode"`
	ContactName     string `json:"contactName"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	Status          string `json:"status"`
	RiskLevel       string `json:"riskLevel"`
	ExpectedVersion int64  `json:"expectedVersion,omitempty"`
}

type createPurchaseOrderBody struct {
	PurchaseRequestID  string `json:"purchaseRequestId"`
	SupplierID         string `json:"supplierId"`
	ExternalReference  string `json:"externalReference"`
	ExpectedDeliveryOn string `json:"expectedDeliveryOn"`
	Note               string `json:"note"`
}

type confirmReceiptBody struct {
	ExpectedVersion  int64  `json:"expectedVersion"`
	ActualDeliveryOn string `json:"actualDeliveryOn"`
}

func (a *api) listSuppliers(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromContext(r.Context())
	if !ok || a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return
	}
	result, err := a.procurement.ListSuppliers(r.Context(), principal)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) createSupplier(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromContext(r.Context())
	if !ok || a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return
	}
	var body supplierBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	supplier, err := a.procurement.CreateSupplier(
		r.Context(), principal, supplierInput(body), correlationIDFromContext(r.Context()),
	)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	w.Header().Set("Location", r.URL.Path+"/"+supplier.ID)
	writeJSON(w, http.StatusCreated, supplier)
}

func (a *api) updateSupplier(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromContext(r.Context())
	if !ok || a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return
	}
	supplierID := strings.TrimSpace(chi.URLParam(r, "supplierID"))
	if !uuidPattern.MatchString(supplierID) {
		writeValidationProblem(w, r, "invalid-supplier-id", "The supplier ID must be a valid UUID.", nil)
		return
	}
	var body supplierBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	supplier, err := a.procurement.UpdateSupplier(
		r.Context(), principal, supplierID,
		procurement.UpdateSupplierInput{SupplierInput: supplierInput(body), ExpectedVersion: body.ExpectedVersion},
		correlationIDFromContext(r.Context()),
	)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, supplier)
}

func (a *api) getOperationsBoard(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromContext(r.Context())
	if !ok || a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return
	}
	result, err := a.procurement.OperationsBoard(r.Context(), principal)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) createPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromContext(r.Context())
	if !ok || a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return
	}
	var body createPurchaseOrderBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	if !uuidPattern.MatchString(strings.TrimSpace(body.PurchaseRequestID)) {
		writeValidationProblem(w, r, "invalid-request-id", "The purchase request ID must be a valid UUID.", nil)
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	order, err := a.procurement.CreatePurchaseOrder(r.Context(), principal, procurement.CreatePurchaseOrderInput{
		PurchaseRequestID:  body.PurchaseRequestID,
		SupplierID:         body.SupplierID,
		ExternalReference:  body.ExternalReference,
		ExpectedDeliveryOn: body.ExpectedDeliveryOn,
		Note:               body.Note,
		IdempotencyKey:     idempotencyKey,
		CorrelationID:      correlationIDFromContext(r.Context()),
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (a *api) confirmPurchaseOrderReceipt(w http.ResponseWriter, r *http.Request) {
	principal, requestID, ok := a.purchaseRequestContext(w, r)
	if !ok {
		return
	}
	var body confirmReceiptBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	order, err := a.procurement.ConfirmReceipt(r.Context(), principal, requestID, procurement.ConfirmReceiptInput{
		ExpectedVersion:  body.ExpectedVersion,
		ActualDeliveryOn: body.ActualDeliveryOn,
		CorrelationID:    correlationIDFromContext(r.Context()),
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func supplierInput(body supplierBody) procurement.SupplierInput {
	return procurement.SupplierInput{
		Code: body.Code, Name: body.Name, TaxCode: body.TaxCode,
		ContactName: body.ContactName, Email: body.Email, Phone: body.Phone,
		Status: body.Status, RiskLevel: body.RiskLevel,
	}
}
