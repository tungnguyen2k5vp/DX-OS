package httpapi

import (
	"net/http"
	"strings"

	"github.com/dx-os-lab/dx-os/backend/internal/procurement"
	"github.com/go-chi/chi/v5"
)

type supplierQuoteBody struct {
	PurchaseRequestID string `json:"purchaseRequestId"`
	SupplierID        string `json:"supplierId"`
	QuoteReference    string `json:"quoteReference"`
	Amount            string `json:"amount"`
	Currency          string `json:"currency"`
	DeliveryOn        string `json:"deliveryOn"`
	WarrantyMonths    int    `json:"warrantyMonths"`
	PaymentTerms      string `json:"paymentTerms"`
	Note              string `json:"note"`
	ExpectedVersion   int64  `json:"expectedVersion"`
}

type selectSupplierQuoteBody struct {
	ExpectedCaseVersion  int64  `json:"expectedCaseVersion"`
	ExpectedQuoteVersion int64  `json:"expectedQuoteVersion"`
	Comment              string `json:"comment"`
}

func (a *api) getSourcingBoard(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	result, err := a.enterprise.SourcingBoard(r.Context(), principal)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) createSupplierQuote(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	var body supplierQuoteBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.enterprise.CreateSupplierQuote(r.Context(), principal, procurement.SupplierQuoteInput{
		PurchaseRequestID: body.PurchaseRequestID, SupplierID: body.SupplierID,
		QuoteReference: body.QuoteReference, Amount: body.Amount, Currency: body.Currency,
		DeliveryOn: body.DeliveryOn, WarrantyMonths: body.WarrantyMonths,
		PaymentTerms: body.PaymentTerms, Note: body.Note,
		IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		CorrelationID:  correlationIDFromContext(r.Context()),
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (a *api) updateSupplierQuote(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "quoteID")
	if !uuidPattern.MatchString(id) {
		writeValidationProblem(w, r, "invalid-quote-id", "Mã báo giá không hợp lệ.", nil)
		return
	}
	var body supplierQuoteBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.enterprise.UpdateSupplierQuote(r.Context(), principal, id, procurement.UpdateSupplierQuoteInput{
		QuoteReference: body.QuoteReference, Amount: body.Amount, Currency: body.Currency,
		DeliveryOn: body.DeliveryOn, WarrantyMonths: body.WarrantyMonths,
		PaymentTerms: body.PaymentTerms, Note: body.Note, ExpectedVersion: body.ExpectedVersion,
		CorrelationID: correlationIDFromContext(r.Context()),
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) selectSupplierQuote(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "quoteID")
	if !uuidPattern.MatchString(id) {
		writeValidationProblem(w, r, "invalid-quote-id", "Mã báo giá không hợp lệ.", nil)
		return
	}
	var body selectSupplierQuoteBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.enterprise.SelectSupplierQuote(r.Context(), principal, id, procurement.SelectSupplierQuoteInput{
		ExpectedCaseVersion: body.ExpectedCaseVersion, ExpectedQuoteVersion: body.ExpectedQuoteVersion,
		Comment: body.Comment, IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		CorrelationID: correlationIDFromContext(r.Context()),
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
