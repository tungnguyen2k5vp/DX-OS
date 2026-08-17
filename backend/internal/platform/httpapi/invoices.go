package httpapi

import (
	"net/http"
	"strings"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/dx-os-lab/dx-os/backend/internal/procurement"
	"github.com/go-chi/chi/v5"
)

type invoiceBody struct {
	PurchaseOrderID string `json:"purchaseOrderId,omitempty"`
	InvoiceNumber   string `json:"invoiceNumber"`
	IssuedOn        string `json:"issuedOn"`
	DueOn           string `json:"dueOn"`
	Amount          string `json:"amount"`
	Currency        string `json:"currency"`
	Note            string `json:"note"`
	ExpectedVersion int64  `json:"expectedVersion,omitempty"`
}

type invoiceActionBody struct {
	Action           string `json:"action"`
	ExpectedVersion  int64  `json:"expectedVersion"`
	Comment          string `json:"comment"`
	PaymentReference string `json:"paymentReference"`
	PaidOn           string `json:"paidOn"`
}

func (a *api) getInvoiceBoard(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromContext(r.Context())
	if !ok || a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return
	}
	result, err := a.procurement.InvoiceBoard(r.Context(), principal)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) createInvoice(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromContext(r.Context())
	if !ok || a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return
	}
	var body invoiceBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.procurement.CreateInvoice(r.Context(), principal, procurement.InvoiceInput{
		PurchaseOrderID: body.PurchaseOrderID, InvoiceNumber: body.InvoiceNumber,
		IssuedOn: body.IssuedOn, DueOn: body.DueOn, Amount: body.Amount,
		Currency: body.Currency, Note: body.Note,
		IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		CorrelationID:  correlationIDFromContext(r.Context()),
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/invoices/"+*result.InvoiceID)
	writeJSON(w, http.StatusCreated, result)
}

func (a *api) updateInvoice(w http.ResponseWriter, r *http.Request) {
	principal, invoiceID, ok := a.invoiceContext(w, r)
	if !ok {
		return
	}
	var body invoiceBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.procurement.UpdateInvoice(r.Context(), principal, invoiceID, procurement.UpdateInvoiceInput{
		InvoiceNumber: body.InvoiceNumber, IssuedOn: body.IssuedOn, DueOn: body.DueOn,
		Amount: body.Amount, Currency: body.Currency, Note: body.Note,
		ExpectedVersion: body.ExpectedVersion, CorrelationID: correlationIDFromContext(r.Context()),
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) transitionInvoice(w http.ResponseWriter, r *http.Request) {
	principal, invoiceID, ok := a.invoiceContext(w, r)
	if !ok {
		return
	}
	var body invoiceActionBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.procurement.TransitionInvoice(r.Context(), principal, invoiceID, procurement.InvoiceActionInput{
		Action: body.Action, ExpectedVersion: body.ExpectedVersion, Comment: body.Comment,
		PaymentReference: body.PaymentReference, PaidOn: body.PaidOn,
		IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		CorrelationID:  correlationIDFromContext(r.Context()),
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) invoiceContext(w http.ResponseWriter, r *http.Request) (auth.Principal, string, bool) {
	principal, ok := principalFromContext(r.Context())
	if !ok || a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return auth.Principal{}, "", false
	}
	invoiceID := strings.TrimSpace(chi.URLParam(r, "invoiceID"))
	if !uuidPattern.MatchString(invoiceID) {
		writeValidationProblem(w, r, "invalid-invoice-id", "The invoice ID must be a valid UUID.", nil)
		return auth.Principal{}, "", false
	}
	return principal, invoiceID, true
}
