package httpapi

import (
	"net/http"
	"strings"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/dx-os-lab/dx-os/backend/internal/procurement"
	"github.com/go-chi/chi/v5"
)

type recordPaymentBody struct {
	ExpectedVersion  int64  `json:"expectedVersion"`
	Amount           string `json:"amount"`
	PaidOn           string `json:"paidOn"`
	PaymentReference string `json:"paymentReference"`
	Note             string `json:"note"`
}

func (a *api) listInvoicePayments(w http.ResponseWriter, r *http.Request) {
	principal, invoiceID, ok := a.enterpriseInvoiceContext(w, r)
	if !ok {
		return
	}
	result, err := a.enterprise.ListInvoicePayments(r.Context(), principal, invoiceID)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) recordInvoicePayment(w http.ResponseWriter, r *http.Request) {
	principal, invoiceID, ok := a.enterpriseInvoiceContext(w, r)
	if !ok {
		return
	}
	var body recordPaymentBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.enterprise.RecordInvoicePayment(r.Context(), principal, invoiceID, procurement.RecordPaymentInput{
		ExpectedVersion: body.ExpectedVersion, Amount: body.Amount, PaidOn: body.PaidOn,
		PaymentReference: body.PaymentReference, Note: body.Note,
		IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		CorrelationID:  correlationIDFromContext(r.Context()),
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (a *api) enterpriseInvoiceContext(w http.ResponseWriter, r *http.Request) (auth.Principal, string, bool) {
	principal, ok := principalFromContext(r.Context())
	if !ok || a.enterprise == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Enterprise procurement service is unavailable.")
		return auth.Principal{}, "", false
	}
	invoiceID := strings.TrimSpace(chi.URLParam(r, "invoiceID"))
	if !uuidPattern.MatchString(invoiceID) {
		writeValidationProblem(w, r, "invalid-invoice-id", "The invoice ID must be a valid UUID.", nil)
		return auth.Principal{}, "", false
	}
	return principal, invoiceID, true
}
