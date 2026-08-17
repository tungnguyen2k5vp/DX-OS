package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/dx-os-lab/dx-os/backend/internal/procurement"
	"github.com/go-chi/chi/v5"
)

const maxRequestBodyBytes = 1 << 20
const maxAttachmentRequestBytes = procurement.MaxAttachmentSize + (1 << 20)

var uuidPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`,
)

type createPurchaseRequestBody struct {
	Title      string                          `json:"title"`
	Reason     string                          `json:"reason"`
	Currency   string                          `json:"currency"`
	CostCenter string                          `json:"costCenter"`
	Items      []createPurchaseRequestItemBody `json:"items"`
}

type createPurchaseRequestItemBody struct {
	Description string `json:"description"`
	Quantity    string `json:"quantity"`
	Unit        string `json:"unit"`
	UnitPrice   string `json:"unitPrice"`
}

type updatePurchaseRequestBody struct {
	Title           string                          `json:"title"`
	Reason          string                          `json:"reason"`
	Currency        string                          `json:"currency"`
	CostCenter      string                          `json:"costCenter"`
	Items           []createPurchaseRequestItemBody `json:"items"`
	ExpectedVersion int64                           `json:"expectedVersion"`
}

type transitionPurchaseRequestBody struct {
	Action          string `json:"action"`
	ExpectedVersion int64  `json:"expectedVersion"`
	Comment         string `json:"comment"`
}

type addPurchaseRequestCommentBody struct {
	Body string `json:"body"`
}

type adjustBudgetAllocationBody struct {
	AllocatedAmount string `json:"allocatedAmount"`
	ExpectedVersion int64  `json:"expectedVersion"`
	Reason          string `json:"reason"`
}

func (a *api) createPurchaseRequest(w http.ResponseWriter, r *http.Request) {
	if a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required", "A valid access token is required.")
		return
	}

	var body createPurchaseRequestBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		var bodyError *requestBodyError
		if errors.As(err, &bodyError) {
			writeProblem(
				w,
				r,
				bodyError.Status,
				bodyError.Code,
				bodyError.Title,
				bodyError.Detail,
			)
			return
		}
		writeProblem(w, r, http.StatusBadRequest, "invalid-request-body", "Invalid request body", "The request body could not be decoded.")
		return
	}

	input := procurement.CreateInput{
		Title:      body.Title,
		Reason:     body.Reason,
		Currency:   body.Currency,
		CostCenter: body.CostCenter,
		Items:      make([]procurement.CreateItemInput, len(body.Items)),
	}
	for index, item := range body.Items {
		input.Items[index] = procurement.CreateItemInput{
			Description: item.Description,
			Quantity:    item.Quantity,
			Unit:        item.Unit,
			UnitPrice:   item.UnitPrice,
		}
	}

	request, err := a.procurement.Create(
		r.Context(),
		principal,
		input,
		correlationIDFromContext(r.Context()),
	)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	w.Header().Set("Location", r.URL.Path+"/"+request.ID)
	writeJSON(w, http.StatusCreated, request)
}

func (a *api) listPurchaseRequests(w http.ResponseWriter, r *http.Request) {
	if a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required", "A valid access token is required.")
		return
	}

	input, violations := parseListInput(r)
	if len(violations) > 0 {
		writeValidationProblem(w, r, "invalid-query", "One or more query parameters are invalid.", violations)
		return
	}
	result, err := a.procurement.List(r.Context(), principal, input)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) getPurchaseRequest(w http.ResponseWriter, r *http.Request) {
	if a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required", "A valid access token is required.")
		return
	}

	requestID := strings.TrimSpace(chi.URLParam(r, "requestID"))
	if !uuidPattern.MatchString(requestID) {
		writeValidationProblem(w, r, "invalid-request-id", "The purchase request ID must be a valid UUID.", nil)
		return
	}
	request, err := a.procurement.Get(r.Context(), principal, requestID)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, request)
}

func (a *api) getPurchaseRequestTimeline(w http.ResponseWriter, r *http.Request) {
	principal, requestID, ok := a.purchaseRequestContext(w, r)
	if !ok {
		return
	}
	input, violations := parseTimelineInput(r)
	if len(violations) > 0 {
		writeValidationProblem(w, r, "invalid-query", "One or more query parameters are invalid.", violations)
		return
	}
	result, err := a.procurement.Timeline(r.Context(), principal, requestID, input)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) listPurchaseRequestComments(w http.ResponseWriter, r *http.Request) {
	principal, requestID, ok := a.purchaseRequestContext(w, r)
	if !ok {
		return
	}
	result, err := a.procurement.ListComments(r.Context(), principal, requestID)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) addPurchaseRequestComment(w http.ResponseWriter, r *http.Request) {
	principal, requestID, ok := a.purchaseRequestContext(w, r)
	if !ok {
		return
	}
	var body addPurchaseRequestCommentBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	comment, err := a.procurement.AddComment(
		r.Context(),
		principal,
		requestID,
		procurement.CommentInput{
			Body:          body.Body,
			CorrelationID: correlationIDFromContext(r.Context()),
		},
	)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, comment)
}

func (a *api) getTaskSummary(w http.ResponseWriter, r *http.Request) {
	if a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required", "A valid access token is required.")
		return
	}
	result, err := a.procurement.TaskSummary(r.Context(), principal)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) getPurchaseRequestBudgetCheck(w http.ResponseWriter, r *http.Request) {
	principal, requestID, ok := a.purchaseRequestContext(w, r)
	if !ok {
		return
	}
	result, err := a.procurement.BudgetCheck(r.Context(), principal, requestID)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) getBudgetSummary(w http.ResponseWriter, r *http.Request) {
	if a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required", "A valid access token is required.")
		return
	}
	query := r.URL.Query()
	var violations []procurement.FieldViolation
	for key, values := range query {
		if key != "costCenter" && key != "currency" {
			violations = append(violations, procurement.FieldViolation{
				Field: key, Message: "is not a supported query parameter",
			})
		}
		if len(values) != 1 {
			violations = append(violations, procurement.FieldViolation{
				Field: key, Message: "must be specified at most once",
			})
		}
	}
	input := procurement.BudgetSummaryInput{
		CostCenter: query.Get("costCenter"),
		Currency:   query.Get("currency"),
	}
	if err := procurement.ValidateBudgetSummaryInput(&input); err != nil {
		var validationError *procurement.ValidationError
		if errors.As(err, &validationError) {
			violations = append(violations, validationError.Violations...)
		}
	}
	if len(violations) > 0 {
		writeValidationProblem(w, r, "invalid-budget-query", "One or more budget query parameters are invalid.", violations)
		return
	}
	result, err := a.procurement.GetBudgetSummary(r.Context(), principal, input)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) getBudgetDashboard(w http.ResponseWriter, r *http.Request) {
	if a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required", "A valid access token is required.")
		return
	}
	result, err := a.procurement.BudgetDashboard(r.Context(), principal)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) adjustBudgetAllocation(w http.ResponseWriter, r *http.Request) {
	if a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required", "A valid access token is required.")
		return
	}
	allocationID := strings.TrimSpace(chi.URLParam(r, "allocationID"))
	if !uuidPattern.MatchString(allocationID) {
		writeValidationProblem(w, r, "invalid-allocation-id", "The budget allocation ID must be a valid UUID.", nil)
		return
	}
	var body adjustBudgetAllocationBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeValidationProblem(w, r, "invalid-idempotency-key", "Idempotency-Key is required for budget adjustments.", []procurement.FieldViolation{
			{Field: "Idempotency-Key", Message: "is required"},
		})
		return
	}
	allocation, err := a.procurement.AdjustBudget(
		r.Context(),
		principal,
		allocationID,
		procurement.AdjustBudgetInput{
			AllocatedAmount: body.AllocatedAmount,
			ExpectedVersion: body.ExpectedVersion,
			Reason:          body.Reason,
			IdempotencyKey:  idempotencyKey,
			CorrelationID:   correlationIDFromContext(r.Context()),
		},
	)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, allocation)
}

func (a *api) updatePurchaseRequest(w http.ResponseWriter, r *http.Request) {
	principal, requestID, ok := a.purchaseRequestContext(w, r)
	if !ok {
		return
	}

	var body updatePurchaseRequestBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	input := procurement.UpdateInput{
		CreateInput: procurement.CreateInput{
			Title:      body.Title,
			Reason:     body.Reason,
			Currency:   body.Currency,
			CostCenter: body.CostCenter,
			Items:      purchaseRequestItems(body.Items),
		},
		ExpectedVersion: body.ExpectedVersion,
	}
	request, err := a.procurement.Update(
		r.Context(),
		principal,
		requestID,
		input,
		correlationIDFromContext(r.Context()),
	)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, request)
}

func (a *api) transitionPurchaseRequest(w http.ResponseWriter, r *http.Request) {
	principal, requestID, ok := a.purchaseRequestContext(w, r)
	if !ok {
		return
	}

	var body transitionPurchaseRequestBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	action, valid := procurement.ParseAction(body.Action)
	if !valid {
		writeValidationProblem(w, r, "invalid-purchase-request-transition", "The transition action is invalid.", []procurement.FieldViolation{
			{Field: "action", Message: "must be a supported purchase request action"},
		})
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeValidationProblem(w, r, "invalid-idempotency-key", "Idempotency-Key is required for purchase request transitions.", []procurement.FieldViolation{
			{Field: "Idempotency-Key", Message: "is required"},
		})
		return
	}
	request, err := a.procurement.Transition(
		r.Context(),
		principal,
		requestID,
		procurement.TransitionInput{
			Action:          action,
			ExpectedVersion: body.ExpectedVersion,
			Comment:         body.Comment,
			IdempotencyKey:  idempotencyKey,
			CorrelationID:   correlationIDFromContext(r.Context()),
		},
	)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, request)
}

func (a *api) listPurchaseRequestAttachments(w http.ResponseWriter, r *http.Request) {
	principal, requestID, ok := a.purchaseRequestContext(w, r)
	if !ok {
		return
	}
	result, err := a.procurement.ListAttachments(r.Context(), principal, requestID)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) uploadPurchaseRequestAttachment(w http.ResponseWriter, r *http.Request) {
	principal, requestID, ok := a.purchaseRequestContext(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentRequestBytes)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writeProblem(
			w, r, http.StatusBadRequest, "invalid-multipart-body",
			"Invalid attachment upload",
			"The request must be multipart/form-data and contain documentType and file.",
		)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeValidationProblem(
			w, r, "invalid-attachment", "An attachment file is required.",
			[]procurement.FieldViolation{{Field: "file", Message: "Tệp đính kèm là bắt buộc."}},
		)
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, procurement.MaxAttachmentSize+1))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid-attachment", "Invalid attachment", "The attachment could not be read.")
		return
	}
	contentType := strings.TrimSpace(header.Header.Get("Content-Type"))
	if mediaType, _, parseErr := mime.ParseMediaType(contentType); parseErr == nil {
		contentType = mediaType
	}
	attachment, err := a.procurement.UploadAttachment(
		r.Context(),
		principal,
		requestID,
		procurement.UploadAttachmentInput{
			DocumentType:  procurement.DocumentType(strings.TrimSpace(r.FormValue("documentType"))),
			FileName:      header.Filename,
			ContentType:   contentType,
			Content:       content,
			CorrelationID: correlationIDFromContext(r.Context()),
		},
	)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	w.Header().Set("Location", r.URL.Path+"/"+attachment.ID+"/content")
	writeJSON(w, http.StatusCreated, attachment)
}

func (a *api) downloadPurchaseRequestAttachment(w http.ResponseWriter, r *http.Request) {
	principal, requestID, attachmentID, ok := a.attachmentContext(w, r)
	if !ok {
		return
	}
	result, err := a.procurement.DownloadAttachment(
		r.Context(), principal, requestID, attachmentID,
	)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	disposition := mime.FormatMediaType(
		"attachment",
		map[string]string{"filename": result.Attachment.FileName},
	)
	w.Header().Set("Content-Type", result.Attachment.ContentType)
	w.Header().Set("Content-Disposition", disposition)
	w.Header().Set("Content-Length", strconv.Itoa(len(result.Content)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(result.Content)
}

func (a *api) deletePurchaseRequestAttachment(w http.ResponseWriter, r *http.Request) {
	principal, requestID, attachmentID, ok := a.attachmentContext(w, r)
	if !ok {
		return
	}
	err := a.procurement.DeleteAttachment(
		r.Context(),
		principal,
		requestID,
		attachmentID,
		correlationIDFromContext(r.Context()),
	)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) attachmentContext(
	w http.ResponseWriter,
	r *http.Request,
) (auth.Principal, string, string, bool) {
	principal, requestID, ok := a.purchaseRequestContext(w, r)
	if !ok {
		return auth.Principal{}, "", "", false
	}
	attachmentID := strings.TrimSpace(chi.URLParam(r, "attachmentID"))
	if !uuidPattern.MatchString(attachmentID) {
		writeValidationProblem(w, r, "invalid-attachment-id", "The attachment ID must be a valid UUID.", nil)
		return auth.Principal{}, "", "", false
	}
	return principal, requestID, attachmentID, true
}

func (a *api) purchaseRequestContext(
	w http.ResponseWriter,
	r *http.Request,
) (auth.Principal, string, bool) {
	if a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return auth.Principal{}, "", false
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required", "A valid access token is required.")
		return auth.Principal{}, "", false
	}
	requestID := strings.TrimSpace(chi.URLParam(r, "requestID"))
	if !uuidPattern.MatchString(requestID) {
		writeValidationProblem(w, r, "invalid-request-id", "The purchase request ID must be a valid UUID.", nil)
		return auth.Principal{}, "", false
	}
	return principal, requestID, true
}

func purchaseRequestItems(body []createPurchaseRequestItemBody) []procurement.CreateItemInput {
	items := make([]procurement.CreateItemInput, len(body))
	for index, item := range body {
		items[index] = procurement.CreateItemInput{
			Description: item.Description,
			Quantity:    item.Quantity,
			Unit:        item.Unit,
			UnitPrice:   item.UnitPrice,
		}
	}
	return items
}

func writeRequestBodyProblem(w http.ResponseWriter, r *http.Request, err error) {
	var bodyError *requestBodyError
	if errors.As(err, &bodyError) {
		writeProblem(w, r, bodyError.Status, bodyError.Code, bodyError.Title, bodyError.Detail)
		return
	}
	writeProblem(w, r, http.StatusBadRequest, "invalid-request-body", "Invalid request body", "The request body could not be decoded.")
}

func (a *api) writeProcurementError(w http.ResponseWriter, r *http.Request, err error) {
	var validationError *procurement.ValidationError
	switch {
	case errors.As(err, &validationError):
		writeValidationProblem(
			w,
			r,
			"invalid-purchase-request",
			"One or more purchase request fields are invalid.",
			validationError.Violations,
		)
	case errors.Is(err, procurement.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", "Forbidden", "The authenticated user cannot perform this operation.")
	case errors.Is(err, procurement.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "purchase-request-not-found", "Purchase request not found", "The purchase request does not exist or is outside the user's data scope.")
	case errors.Is(err, procurement.ErrVersionConflict):
		writeProblem(w, r, http.StatusConflict, "purchase-request-version-conflict", "Purchase request changed", "Reload the latest purchase request before trying this operation again.")
	case errors.Is(err, procurement.ErrIdempotencyConflict):
		writeProblem(w, r, http.StatusConflict, "idempotency-key-conflict", "Idempotency key conflict", "This Idempotency-Key has already been used for another operation.")
	case errors.Is(err, procurement.ErrInvalidTransition):
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid-purchase-request-transition", "Invalid purchase request transition", "The requested action is not allowed in the current status.")
	case errors.Is(err, procurement.ErrBudgetNotFound):
		writeProblem(w, r, http.StatusNotFound, "budget-not-found", "Budget not found", "No active budget allocation matches the requested cost center and currency.")
	case errors.Is(err, procurement.ErrBudgetNotConfigured):
		writeProblem(w, r, http.StatusConflict, "budget-not-configured", "Budget is not configured", "Manager approval requires an active budget allocation for this cost center and currency.")
	case errors.Is(err, procurement.ErrInsufficientBudget):
		writeProblem(w, r, http.StatusConflict, "insufficient-budget", "Insufficient budget", "The available budget cannot cover this purchase request.")
	case errors.Is(err, procurement.ErrBudgetReservation):
		writeProblem(w, r, http.StatusConflict, "budget-reservation-conflict", "Budget reservation conflict", "The purchase request does not have the budget reservation required for this transition.")
	case errors.Is(err, procurement.ErrBudgetVersionConflict):
		writeProblem(w, r, http.StatusConflict, "budget-version-conflict", "Budget allocation changed", "Reload the latest budget dashboard before adjusting this allocation.")
	case errors.Is(err, procurement.ErrBudgetBelowUsage):
		writeProblem(w, r, http.StatusConflict, "budget-below-usage", "Budget is below current usage", "The allocation cannot be lower than the amount already reserved and committed.")
	case errors.Is(err, procurement.ErrAttachmentNotFound):
		writeProblem(w, r, http.StatusNotFound, "attachment-not-found", "Attachment not found", "The attachment does not exist or is outside the purchase request.")
	case errors.Is(err, procurement.ErrAttachmentRequired):
		writeProblem(w, r, http.StatusUnprocessableEntity, "quotation-required", "Quotation required", "Upload a quotation before submitting this purchase request.")
	case errors.Is(err, procurement.ErrDocumentStore):
		writeProblem(w, r, http.StatusServiceUnavailable, "document-store-unavailable", "Document store unavailable", "The attachment service is temporarily unavailable. Please try again.")
	case errors.Is(err, procurement.ErrSupplierNotFound):
		writeProblem(w, r, http.StatusNotFound, "supplier-not-found", "Supplier not found", "The supplier does not exist or is outside the organization scope.")
	case errors.Is(err, procurement.ErrSupplierConflict):
		writeProblem(w, r, http.StatusConflict, "supplier-conflict", "Supplier conflict", "The supplier code or tax code already exists.")
	case errors.Is(err, procurement.ErrSupplierVersion):
		writeProblem(w, r, http.StatusConflict, "supplier-version-conflict", "Supplier changed", "Reload the supplier before updating it.")
	case errors.Is(err, procurement.ErrPurchaseOrderNotFound):
		writeProblem(w, r, http.StatusNotFound, "purchase-order-not-found", "Purchase order not found", "No purchase order exists for this request in the current scope.")
	case errors.Is(err, procurement.ErrPurchaseOrderConflict):
		writeProblem(w, r, http.StatusConflict, "purchase-order-conflict", "Purchase order conflict", "The request already has an order or the idempotency key was used elsewhere.")
	case errors.Is(err, procurement.ErrInvalidFulfillment):
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid-fulfillment-operation", "Invalid fulfillment operation", "The supplier, request status, or delivery state does not allow this operation.")
	case errors.Is(err, procurement.ErrInvoiceNotFound):
		writeProblem(w, r, http.StatusNotFound, "invoice-not-found", "Invoice not found", "The invoice does not exist or is outside the organization scope.")
	case errors.Is(err, procurement.ErrInvoiceConflict):
		writeProblem(w, r, http.StatusConflict, "invoice-conflict", "Invoice conflict", "The order already has an invoice, the invoice number exists, or the idempotency key conflicts.")
	case errors.Is(err, procurement.ErrInvoiceVersion):
		writeProblem(w, r, http.StatusConflict, "invoice-version-conflict", "Invoice changed", "Reload the invoice before trying this operation again.")
	case errors.Is(err, procurement.ErrInvalidInvoiceAction):
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid-invoice-action", "Invalid invoice action", "The requested action is not allowed in the current invoice status.")
	case errors.Is(err, procurement.ErrInvoiceMismatch):
		writeProblem(w, r, http.StatusConflict, "invoice-mismatch", "Invoice does not match", "Verification requires a received order with matching amount and currency.")
	case errors.Is(err, procurement.ErrPolicyNotFound):
		writeProblem(w, r, http.StatusNotFound, "policy-not-found", "Policy not found", "The operating policy does not exist in this organization.")
	case errors.Is(err, procurement.ErrPolicyVersion):
		writeProblem(w, r, http.StatusConflict, "policy-version-conflict", "Policy changed", "Reload the policy center before updating this policy.")
	case errors.Is(err, procurement.ErrAuditCaseNotFound):
		writeProblem(w, r, http.StatusNotFound, "audit-case-not-found", "Audit case not found", "The audit case does not exist or is outside the organization scope.")
	case errors.Is(err, procurement.ErrAuditCaseVersion):
		writeProblem(w, r, http.StatusConflict, "audit-case-version-conflict", "Audit case changed", "Reload the audit case before updating it.")
	case errors.Is(err, procurement.ErrAdminUserNotFound):
		writeProblem(w, r, http.StatusNotFound, "admin-user-not-found", "User not found", "The business user does not exist in this organization.")
	case errors.Is(err, procurement.ErrAdminDepartmentNotFound):
		writeProblem(w, r, http.StatusNotFound, "admin-department-not-found", "Department not found", "The department does not exist, is inactive, or is outside the organization.")
	case errors.Is(err, procurement.ErrAdminVersion):
		writeProblem(w, r, http.StatusConflict, "admin-version-conflict", "Administrative resource changed", "Reload the administration center before saving again.")
	case errors.Is(err, procurement.ErrAdminConflict):
		writeProblem(w, r, http.StatusConflict, "admin-resource-conflict", "Administrative change conflicts", "The change would duplicate a code, deactivate an in-use department, deactivate your own account, or create an invalid hierarchy.")
	case errors.Is(err, procurement.ErrAIRecommendationNotFound):
		writeProblem(w, r, http.StatusNotFound, "ai-recommendation-not-found", "Recommendation not found", "The recommendation does not exist or is outside the organization scope.")
	case errors.Is(err, procurement.ErrAIRecommendationVersion):
		writeProblem(w, r, http.StatusConflict, "ai-recommendation-version-conflict", "Recommendation changed", "Reload the recommendation center before deciding again.")
	case errors.Is(err, procurement.ErrInvalidAIAction):
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid-ai-recommendation-action", "Invalid recommendation action", "Only pending recommendations can be decided.")
	default:
		a.logger.Error(
			"procurement request failed",
			"error", err,
			"correlation_id", correlationIDFromContext(r.Context()),
		)
		writeProblem(w, r, http.StatusInternalServerError, "internal", "Internal server error", "The procurement operation could not be completed.")
	}
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, destination any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return &requestBodyError{
			Status: http.StatusUnsupportedMediaType,
			Code:   "unsupported-media-type",
			Title:  "Unsupported media type",
			Detail: "Content-Type must be application/json.",
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(destination); err != nil {
		if errors.Is(err, io.EOF) {
			return invalidBody("The request body must contain one JSON object.")
		}
		return invalidBody("The request body must be valid JSON and contain only supported fields.")
	}
	if err = decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return invalidBody("The request body must contain exactly one JSON object.")
	}
	return nil
}

type requestBodyError struct {
	Status int
	Code   string
	Title  string
	Detail string
}

func (e *requestBodyError) Error() string {
	return e.Detail
}

func invalidBody(detail string) error {
	return &requestBodyError{
		Status: http.StatusBadRequest,
		Code:   "invalid-request-body",
		Title:  "Invalid request body",
		Detail: detail,
	}
}

func parseListInput(r *http.Request) (procurement.ListInput, []procurement.FieldViolation) {
	input := procurement.ListInput{Page: 1, PageSize: 20, Sort: "createdAt", Direction: "desc"}
	var violations []procurement.FieldViolation
	query := r.URL.Query()
	supported := map[string]bool{
		"page": true, "pageSize": true, "status": true, "search": true,
		"department": true, "costCenter": true, "requester": true,
		"from": true, "to": true, "minAmount": true, "maxAmount": true,
		"sort": true, "direction": true,
	}

	for key, values := range query {
		if !supported[key] {
			violations = append(violations, procurement.FieldViolation{Field: key, Message: "is not a supported query parameter"})
		}
		if len(values) != 1 {
			violations = append(violations, procurement.FieldViolation{Field: key, Message: "must be specified at most once"})
		}
	}

	if value := query.Get("page"); value != "" {
		page, err := strconv.Atoi(value)
		if err != nil || page < 1 {
			violations = append(violations, procurement.FieldViolation{Field: "page", Message: "must be an integer greater than or equal to 1"})
		} else {
			input.Page = page
		}
	}
	if value := query.Get("pageSize"); value != "" {
		pageSize, err := strconv.Atoi(value)
		if err != nil || pageSize < 1 || pageSize > 100 {
			violations = append(violations, procurement.FieldViolation{Field: "pageSize", Message: "must be an integer between 1 and 100"})
		} else {
			input.PageSize = pageSize
		}
	}
	if value := query.Get("status"); value != "" {
		status, valid := procurement.ParseStatus(value)
		if !valid {
			violations = append(violations, procurement.FieldViolation{Field: "status", Message: "must be a supported purchase request status"})
		} else {
			input.Status = &status
		}
	}
	input.Search = strings.TrimSpace(query.Get("search"))
	input.Department = strings.TrimSpace(query.Get("department"))
	input.CostCenter = strings.TrimSpace(query.Get("costCenter"))
	input.Requester = strings.TrimSpace(query.Get("requester"))
	for field, value := range map[string]string{
		"search": input.Search, "department": input.Department,
		"costCenter": input.CostCenter, "requester": input.Requester,
	} {
		if len([]rune(value)) > 100 {
			violations = append(violations, procurement.FieldViolation{Field: field, Message: "must not exceed 100 characters"})
		}
	}
	input.From = strings.TrimSpace(query.Get("from"))
	input.To = strings.TrimSpace(query.Get("to"))
	for field, value := range map[string]string{"from": input.From, "to": input.To} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				violations = append(violations, procurement.FieldViolation{Field: field, Message: "must use YYYY-MM-DD format"})
			}
		}
	}
	input.MinAmount = strings.TrimSpace(query.Get("minAmount"))
	input.MaxAmount = strings.TrimSpace(query.Get("maxAmount"))
	for field, value := range map[string]string{"minAmount": input.MinAmount, "maxAmount": input.MaxAmount} {
		if value != "" && !regexp.MustCompile(`^(0|[1-9][0-9]{0,14})(\.[0-9]{1,4})?$`).MatchString(value) {
			violations = append(violations, procurement.FieldViolation{Field: field, Message: "must be a non-negative monetary amount"})
		}
	}
	if input.MinAmount != "" && input.MaxAmount != "" {
		minAmount, _ := new(big.Rat).SetString(input.MinAmount)
		maxAmount, _ := new(big.Rat).SetString(input.MaxAmount)
		if minAmount != nil && maxAmount != nil && minAmount.Cmp(maxAmount) > 0 {
			violations = append(violations, procurement.FieldViolation{Field: "minAmount", Message: "must not exceed maxAmount"})
		}
	}
	if value := strings.TrimSpace(query.Get("sort")); value != "" {
		if value != "createdAt" && value != "updatedAt" && value != "amount" && value != "code" {
			violations = append(violations, procurement.FieldViolation{Field: "sort", Message: "must be createdAt, updatedAt, amount or code"})
		} else {
			input.Sort = value
		}
	}
	if value := strings.ToLower(strings.TrimSpace(query.Get("direction"))); value != "" {
		if value != "asc" && value != "desc" {
			violations = append(violations, procurement.FieldViolation{Field: "direction", Message: "must be asc or desc"})
		} else {
			input.Direction = value
		}
	}
	return input, violations
}

func parseTimelineInput(r *http.Request) (procurement.TimelineInput, []procurement.FieldViolation) {
	input := procurement.TimelineInput{Page: 1, PageSize: 20}
	var violations []procurement.FieldViolation
	query := r.URL.Query()

	for key, values := range query {
		if key != "page" && key != "pageSize" {
			violations = append(violations, procurement.FieldViolation{
				Field: key, Message: "is not a supported query parameter",
			})
		}
		if len(values) != 1 {
			violations = append(violations, procurement.FieldViolation{
				Field: key, Message: "must be specified at most once",
			})
		}
	}
	if value := query.Get("page"); value != "" {
		page, err := strconv.Atoi(value)
		if err != nil || page < 1 {
			violations = append(violations, procurement.FieldViolation{
				Field: "page", Message: "must be an integer greater than or equal to 1",
			})
		} else {
			input.Page = page
		}
	}
	if value := query.Get("pageSize"); value != "" {
		pageSize, err := strconv.Atoi(value)
		if err != nil || pageSize < 1 || pageSize > 100 {
			violations = append(violations, procurement.FieldViolation{
				Field: "pageSize", Message: "must be an integer between 1 and 100",
			})
		} else {
			input.PageSize = pageSize
		}
	}
	return input, violations
}
