package httpapi

import (
	"net/http"
	"strings"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/dx-os-lab/dx-os/backend/internal/procurement"
	"github.com/go-chi/chi/v5"
)

type receiptItemBody struct {
	PurchaseRequestItemID string `json:"purchaseRequestItemId"`
	QuantityReceived      string `json:"quantityReceived"`
	Condition             string `json:"condition"`
	Note                  string `json:"note"`
}

type recordReceiptBody struct {
	ExpectedVersion int64             `json:"expectedVersion"`
	Outcome         string            `json:"outcome"`
	ReceivedOn      string            `json:"receivedOn"`
	Note            string            `json:"note"`
	Items           []receiptItemBody `json:"items"`
}

type updateOrderBody struct {
	SupplierID         string `json:"supplierId"`
	ExternalReference  string `json:"externalReference"`
	ExpectedDeliveryOn string `json:"expectedDeliveryOn"`
	Note               string `json:"note"`
	ExpectedVersion    int64  `json:"expectedVersion"`
}

type orderTransitionBody struct {
	Action          string `json:"action"`
	ExpectedVersion int64  `json:"expectedVersion"`
	Reason          string `json:"reason"`
}

func (a *api) listPurchaseOrderReceipts(w http.ResponseWriter, r *http.Request) {
	principal, requestID, ok := a.enterpriseRequestContext(w, r)
	if !ok {
		return
	}
	result, err := a.enterprise.ListReceipts(r.Context(), principal, requestID)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) recordPurchaseOrderReceipt(w http.ResponseWriter, r *http.Request) {
	principal, requestID, ok := a.enterpriseRequestContext(w, r)
	if !ok {
		return
	}
	var body recordReceiptBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	items := make([]procurement.ReceiptItemInput, len(body.Items))
	for index, item := range body.Items {
		items[index] = procurement.ReceiptItemInput{PurchaseRequestItemID: item.PurchaseRequestItemID, QuantityReceived: item.QuantityReceived, Condition: item.Condition, Note: item.Note}
	}
	result, err := a.enterprise.RecordReceipt(r.Context(), principal, requestID, procurement.RecordReceiptInput{
		ExpectedVersion: body.ExpectedVersion, Outcome: body.Outcome, ReceivedOn: body.ReceivedOn,
		Note: body.Note, Items: items, IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		CorrelationID: correlationIDFromContext(r.Context()),
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (a *api) updatePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	principal, requestID, ok := a.enterpriseRequestContext(w, r)
	if !ok {
		return
	}
	var body updateOrderBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.enterprise.UpdatePurchaseOrder(r.Context(), principal, requestID, procurement.UpdatePurchaseOrderInput{
		SupplierID: body.SupplierID, ExternalReference: body.ExternalReference,
		ExpectedDeliveryOn: body.ExpectedDeliveryOn, Note: body.Note,
		ExpectedVersion: body.ExpectedVersion, CorrelationID: correlationIDFromContext(r.Context()),
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) transitionPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	principal, requestID, ok := a.enterpriseRequestContext(w, r)
	if !ok {
		return
	}
	var body orderTransitionBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	if strings.ToUpper(strings.TrimSpace(body.Action)) != "CANCEL" {
		writeValidationProblem(w, r, "invalid-order-action", "Endpoint này chỉ hỗ trợ thao tác CANCEL (hủy đơn).", nil)
		return
	}
	result, err := a.enterprise.CancelPurchaseOrder(r.Context(), principal, requestID, procurement.CancelPurchaseOrderInput{
		ExpectedVersion: body.ExpectedVersion, Reason: body.Reason,
		IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		CorrelationID:  correlationIDFromContext(r.Context()),
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) enterpriseRequestContext(w http.ResponseWriter, r *http.Request) (auth.Principal, string, bool) {
	principal, ok := principalFromContext(r.Context())
	if !ok || a.enterprise == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Dịch vụ chưa sẵn sàng", "Dịch vụ mua sắm doanh nghiệp hiện không sẵn sàng.")
		return auth.Principal{}, "", false
	}
	requestID := strings.TrimSpace(chi.URLParam(r, "requestID"))
	if !uuidPattern.MatchString(requestID) {
		writeValidationProblem(w, r, "invalid-request-id", "Mã phiếu mua sắm phải là UUID hợp lệ.", nil)
		return auth.Principal{}, "", false
	}
	return principal, requestID, true
}
