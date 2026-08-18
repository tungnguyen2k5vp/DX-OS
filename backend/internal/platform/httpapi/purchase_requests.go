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
const maxAttachmentMultipartParts = 10
const maxAttachmentDocumentTypeBytes = 128

var errAttachmentFileRequired = errors.New("attachment file is required")

type attachmentUpload struct {
	documentType string
	fileName     string
	contentType  string
	content      []byte
}

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
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Dịch vụ chưa sẵn sàng", "Dịch vụ mua sắm hiện không sẵn sàng.")
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Cần access token hợp lệ để tiếp tục.")
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
		writeProblem(w, r, http.StatusBadRequest, "invalid-request-body", "Nội dung yêu cầu không hợp lệ", "Không thể đọc nội dung yêu cầu.")
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
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Dịch vụ chưa sẵn sàng", "Dịch vụ mua sắm hiện không sẵn sàng.")
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Cần access token hợp lệ để tiếp tục.")
		return
	}

	input, violations := parseListInput(r)
	if len(violations) > 0 {
		writeValidationProblem(w, r, "invalid-query", "Một hoặc nhiều tham số truy vấn không hợp lệ.", violations)
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
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Dịch vụ chưa sẵn sàng", "Dịch vụ mua sắm hiện không sẵn sàng.")
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Cần access token hợp lệ để tiếp tục.")
		return
	}

	requestID := strings.TrimSpace(chi.URLParam(r, "requestID"))
	if !uuidPattern.MatchString(requestID) {
		writeValidationProblem(w, r, "invalid-request-id", "Mã phiếu mua sắm phải là UUID hợp lệ.", nil)
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
		writeValidationProblem(w, r, "invalid-query", "Một hoặc nhiều tham số truy vấn không hợp lệ.", violations)
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
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Dịch vụ chưa sẵn sàng", "Dịch vụ mua sắm hiện không sẵn sàng.")
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Cần access token hợp lệ để tiếp tục.")
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
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Dịch vụ chưa sẵn sàng", "Dịch vụ mua sắm hiện không sẵn sàng.")
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Cần access token hợp lệ để tiếp tục.")
		return
	}
	query := r.URL.Query()
	var violations []procurement.FieldViolation
	for key, values := range query {
		if key != "costCenter" && key != "currency" {
			violations = append(violations, procurement.FieldViolation{
				Field: key, Message: "Tham số truy vấn này không được hỗ trợ.",
			})
		}
		if len(values) != 1 {
			violations = append(violations, procurement.FieldViolation{
				Field: key, Message: "Chỉ được chỉ định tối đa một lần.",
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
		writeValidationProblem(w, r, "invalid-budget-query", "Một hoặc nhiều tham số truy vấn ngân sách không hợp lệ.", violations)
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
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Dịch vụ chưa sẵn sàng", "Dịch vụ mua sắm hiện không sẵn sàng.")
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Cần access token hợp lệ để tiếp tục.")
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
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Dịch vụ chưa sẵn sàng", "Dịch vụ mua sắm hiện không sẵn sàng.")
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Cần access token hợp lệ để tiếp tục.")
		return
	}
	allocationID := strings.TrimSpace(chi.URLParam(r, "allocationID"))
	if !uuidPattern.MatchString(allocationID) {
		writeValidationProblem(w, r, "invalid-allocation-id", "Mã hạn mức ngân sách phải là UUID hợp lệ.", nil)
		return
	}
	var body adjustBudgetAllocationBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeValidationProblem(w, r, "invalid-idempotency-key", "Cần Idempotency-Key khi điều chỉnh ngân sách.", []procurement.FieldViolation{
			{Field: "Idempotency-Key", Message: "Trường này là bắt buộc."},
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
		writeValidationProblem(w, r, "invalid-purchase-request-transition", "Thao tác chuyển trạng thái không hợp lệ.", []procurement.FieldViolation{
			{Field: "action", Message: "Phải là thao tác phiếu mua sắm được hỗ trợ."},
		})
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeValidationProblem(w, r, "invalid-idempotency-key", "Cần Idempotency-Key khi chuyển trạng thái phiếu mua sắm.", []procurement.FieldViolation{
			{Field: "Idempotency-Key", Message: "Trường này là bắt buộc."},
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
	upload, err := parseAttachmentUpload(r)
	if errors.Is(err, errAttachmentFileRequired) {
		writeValidationProblem(
			w, r, "invalid-attachment", "Tệp đính kèm là bắt buộc.",
			[]procurement.FieldViolation{{Field: "file", Message: "Tệp đính kèm là bắt buộc."}},
		)
		return
	}
	if err != nil {
		writeProblem(
			w, r, http.StatusBadRequest, "invalid-multipart-body",
			"Tải tệp đính kèm không hợp lệ",
			"Yêu cầu phải có dạng multipart/form-data và chứa documentType cùng file.",
		)
		return
	}
	contentType := strings.TrimSpace(upload.contentType)
	if mediaType, _, parseErr := mime.ParseMediaType(contentType); parseErr == nil {
		contentType = mediaType
	}
	attachment, err := a.procurement.UploadAttachment(
		r.Context(),
		principal,
		requestID,
		procurement.UploadAttachmentInput{
			DocumentType:  procurement.DocumentType(strings.TrimSpace(upload.documentType)),
			FileName:      upload.fileName,
			ContentType:   contentType,
			Content:       upload.content,
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

func parseAttachmentUpload(r *http.Request) (attachmentUpload, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return attachmentUpload{}, err
	}
	var result attachmentUpload
	fileFound := false
	partCount := 0
	for {
		part, nextErr := reader.NextPart()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return attachmentUpload{}, nextErr
		}
		partCount++
		if partCount > maxAttachmentMultipartParts {
			_ = part.Close()
			return attachmentUpload{}, errors.New("multipart body contains too many parts")
		}

		switch part.FormName() {
		case "documentType":
			value, readErr := io.ReadAll(io.LimitReader(part, maxAttachmentDocumentTypeBytes+1))
			closeErr := part.Close()
			if readErr != nil || closeErr != nil || len(value) > maxAttachmentDocumentTypeBytes {
				return attachmentUpload{}, errors.New("invalid documentType multipart field")
			}
			result.documentType = string(value)
		case "file":
			if fileFound || strings.TrimSpace(part.FileName()) == "" {
				_ = part.Close()
				return attachmentUpload{}, errors.New("multipart body must contain exactly one named file")
			}
			content, readErr := io.ReadAll(io.LimitReader(part, procurement.MaxAttachmentSize+1))
			closeErr := part.Close()
			if readErr != nil || closeErr != nil {
				return attachmentUpload{}, errors.New("attachment part could not be read")
			}
			result.fileName = part.FileName()
			result.contentType = part.Header.Get("Content-Type")
			result.content = content
			fileFound = true
		default:
			_, readErr := io.Copy(io.Discard, io.LimitReader(part, maxAttachmentRequestBytes+1))
			closeErr := part.Close()
			if readErr != nil || closeErr != nil {
				return attachmentUpload{}, errors.New("multipart part could not be discarded")
			}
		}
	}
	if !fileFound {
		return attachmentUpload{}, errAttachmentFileRequired
	}
	return result, nil
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
		writeValidationProblem(w, r, "invalid-attachment-id", "Mã tệp đính kèm phải là UUID hợp lệ.", nil)
		return auth.Principal{}, "", "", false
	}
	return principal, requestID, attachmentID, true
}

func (a *api) purchaseRequestContext(
	w http.ResponseWriter,
	r *http.Request,
) (auth.Principal, string, bool) {
	if a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Dịch vụ chưa sẵn sàng", "Dịch vụ mua sắm hiện không sẵn sàng.")
		return auth.Principal{}, "", false
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Cần access token hợp lệ để tiếp tục.")
		return auth.Principal{}, "", false
	}
	requestID := strings.TrimSpace(chi.URLParam(r, "requestID"))
	if !uuidPattern.MatchString(requestID) {
		writeValidationProblem(w, r, "invalid-request-id", "Mã phiếu mua sắm phải là UUID hợp lệ.", nil)
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
	writeProblem(w, r, http.StatusBadRequest, "invalid-request-body", "Nội dung yêu cầu không hợp lệ", "Không thể đọc nội dung yêu cầu.")
}

func (a *api) writeProcurementError(w http.ResponseWriter, r *http.Request, err error) {
	var validationError *procurement.ValidationError
	switch {
	case errors.As(err, &validationError):
		writeValidationProblem(
			w,
			r,
			"invalid-purchase-request",
			"Một hoặc nhiều trường của phiếu mua sắm không hợp lệ.",
			validationError.Violations,
		)
	case errors.Is(err, procurement.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", "Không có quyền thực hiện", "Tài khoản hiện tại không được phép thực hiện thao tác này.")
	case errors.Is(err, procurement.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "purchase-request-not-found", "Không tìm thấy phiếu mua sắm", "Phiếu không tồn tại hoặc nằm ngoài phạm vi dữ liệu của tài khoản.")
	case errors.Is(err, procurement.ErrVersionConflict):
		writeProblem(w, r, http.StatusConflict, "purchase-request-version-conflict", "Phiếu mua sắm đã thay đổi", "Hãy tải lại phiên bản mới nhất của phiếu trước khi thử lại.")
	case errors.Is(err, procurement.ErrIdempotencyConflict):
		writeProblem(w, r, http.StatusConflict, "idempotency-key-conflict", "Idempotency-Key bị trùng", "Idempotency-Key này đã được dùng cho thao tác khác.")
	case errors.Is(err, procurement.ErrInvalidTransition):
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid-purchase-request-transition", "Không thể chuyển trạng thái phiếu", "Trạng thái hiện tại không cho phép thực hiện thao tác đã chọn.")
	case errors.Is(err, procurement.ErrBudgetNotFound):
		writeProblem(w, r, http.StatusNotFound, "budget-not-found", "Không tìm thấy ngân sách", "Không có hạn mức ngân sách đang hoạt động phù hợp với trung tâm chi phí và tiền tệ của phiếu.")
	case errors.Is(err, procurement.ErrBudgetNotConfigured):
		writeProblem(w, r, http.StatusConflict, "budget-not-configured", "Chưa cấu hình ngân sách", "Trưởng bộ phận chỉ có thể phê duyệt khi trung tâm chi phí và tiền tệ này có hạn mức ngân sách đang hoạt động.")
	case errors.Is(err, procurement.ErrInsufficientBudget):
		writeProblem(w, r, http.StatusConflict, "insufficient-budget", "Ngân sách không đủ", "Ngân sách khả dụng không đủ để chi trả cho phiếu mua sắm này.")
	case errors.Is(err, procurement.ErrBudgetReservation):
		writeProblem(w, r, http.StatusConflict, "budget-reservation-conflict", "Không có khoản giữ ngân sách phù hợp", "Phiếu chưa có khoản giữ ngân sách cần thiết để chuyển sang trạng thái này.")
	case errors.Is(err, procurement.ErrBudgetVersionConflict):
		writeProblem(w, r, http.StatusConflict, "budget-version-conflict", "Hạn mức ngân sách đã thay đổi", "Hãy tải lại bảng điều khiển ngân sách trước khi điều chỉnh hạn mức này.")
	case errors.Is(err, procurement.ErrBudgetBelowUsage):
		writeProblem(w, r, http.StatusConflict, "budget-below-usage", "Hạn mức thấp hơn số tiền đã sử dụng", "Hạn mức không được thấp hơn tổng số tiền đang giữ và đã cam kết.")
	case errors.Is(err, procurement.ErrAttachmentNotFound):
		writeProblem(w, r, http.StatusNotFound, "attachment-not-found", "Không tìm thấy tệp đính kèm", "Tệp không tồn tại hoặc không thuộc phiếu mua sắm này.")
	case errors.Is(err, procurement.ErrAttachmentRequired):
		writeProblem(w, r, http.StatusUnprocessableEntity, "quotation-required", "Cần tài liệu báo giá", "Hãy tải lên báo giá trước khi gửi phiếu mua sắm.")
	case errors.Is(err, procurement.ErrDocumentStore):
		writeProblem(w, r, http.StatusServiceUnavailable, "document-store-unavailable", "Kho tài liệu chưa sẵn sàng", "Dịch vụ tệp đính kèm đang tạm thời gián đoạn. Vui lòng thử lại.")
	case errors.Is(err, procurement.ErrSupplierNotFound):
		writeProblem(w, r, http.StatusNotFound, "supplier-not-found", "Không tìm thấy nhà cung cấp", "Nhà cung cấp không tồn tại hoặc nằm ngoài phạm vi tổ chức.")
	case errors.Is(err, procurement.ErrSupplierConflict):
		writeProblem(w, r, http.StatusConflict, "supplier-conflict", "Thông tin nhà cung cấp bị trùng", "Mã nhà cung cấp hoặc mã số thuế đã tồn tại.")
	case errors.Is(err, procurement.ErrSupplierVersion):
		writeProblem(w, r, http.StatusConflict, "supplier-version-conflict", "Nhà cung cấp đã thay đổi", "Hãy tải lại thông tin nhà cung cấp trước khi cập nhật.")
	case errors.Is(err, procurement.ErrPurchaseOrderNotFound):
		writeProblem(w, r, http.StatusNotFound, "purchase-order-not-found", "Không tìm thấy đơn hàng", "Không có đơn hàng cho phiếu này trong phạm vi hiện tại.")
	case errors.Is(err, procurement.ErrPurchaseOrderConflict):
		writeProblem(w, r, http.StatusConflict, "purchase-order-conflict", "Xung đột đơn hàng", "Phiếu đã có đơn hàng hoặc Idempotency-Key đã được dùng ở nơi khác.")
	case errors.Is(err, procurement.ErrInvalidFulfillment):
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid-fulfillment-operation", "Thao tác giao nhận không hợp lệ", "Nhà cung cấp, trạng thái phiếu hoặc trạng thái giao hàng không cho phép thao tác này.")
	case errors.Is(err, procurement.ErrInvoiceNotFound):
		writeProblem(w, r, http.StatusNotFound, "invoice-not-found", "Không tìm thấy hóa đơn", "Hóa đơn không tồn tại hoặc nằm ngoài phạm vi tổ chức.")
	case errors.Is(err, procurement.ErrInvoiceConflict):
		writeProblem(w, r, http.StatusConflict, "invoice-conflict", "Xung đột hóa đơn", "Đơn hàng đã có hóa đơn, số hóa đơn đã tồn tại hoặc Idempotency-Key bị trùng.")
	case errors.Is(err, procurement.ErrInvoiceVersion):
		writeProblem(w, r, http.StatusConflict, "invoice-version-conflict", "Hóa đơn đã thay đổi", "Hãy tải lại hóa đơn trước khi thực hiện lại thao tác.")
	case errors.Is(err, procurement.ErrInvalidInvoiceAction):
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid-invoice-action", "Thao tác hóa đơn không hợp lệ", "Trạng thái hiện tại của hóa đơn không cho phép thao tác đã chọn.")
	case errors.Is(err, procurement.ErrInvoiceMismatch):
		writeProblem(w, r, http.StatusConflict, "invoice-mismatch", "Hóa đơn không khớp", "Chỉ có thể xác minh khi đơn hàng đã được nhận và số tiền, tiền tệ khớp nhau.")
	case errors.Is(err, procurement.ErrPolicyNotFound):
		writeProblem(w, r, http.StatusNotFound, "policy-not-found", "Không tìm thấy quy tắc vận hành", "Quy tắc vận hành không tồn tại trong tổ chức này.")
	case errors.Is(err, procurement.ErrPolicyVersion):
		writeProblem(w, r, http.StatusConflict, "policy-version-conflict", "Quy tắc đã thay đổi", "Hãy tải lại trung tâm chính sách trước khi cập nhật quy tắc này.")
	case errors.Is(err, procurement.ErrAuditCaseNotFound):
		writeProblem(w, r, http.StatusNotFound, "audit-case-not-found", "Không tìm thấy hồ sơ kiểm toán", "Hồ sơ kiểm toán không tồn tại hoặc nằm ngoài phạm vi tổ chức.")
	case errors.Is(err, procurement.ErrAuditCaseVersion):
		writeProblem(w, r, http.StatusConflict, "audit-case-version-conflict", "Hồ sơ kiểm toán đã thay đổi", "Hãy tải lại hồ sơ kiểm toán trước khi cập nhật.")
	case errors.Is(err, procurement.ErrAdminUserNotFound):
		writeProblem(w, r, http.StatusNotFound, "admin-user-not-found", "Không tìm thấy người dùng", "Người dùng nghiệp vụ không tồn tại trong tổ chức này.")
	case errors.Is(err, procurement.ErrAdminDepartmentNotFound):
		writeProblem(w, r, http.StatusNotFound, "admin-department-not-found", "Không tìm thấy phòng ban", "Phòng ban không tồn tại, đã ngừng hoạt động hoặc nằm ngoài tổ chức.")
	case errors.Is(err, procurement.ErrAdminVersion):
		writeProblem(w, r, http.StatusConflict, "admin-version-conflict", "Dữ liệu quản trị đã thay đổi", "Hãy tải lại trung tâm quản trị trước khi lưu lại.")
	case errors.Is(err, procurement.ErrAdminConflict):
		writeProblem(w, r, http.StatusConflict, "admin-resource-conflict", "Thay đổi quản trị bị xung đột", "Thay đổi có thể làm trùng mã, vô hiệu hóa phòng ban đang được sử dụng, vô hiệu hóa chính tài khoản của bạn hoặc tạo cây phòng ban không hợp lệ.")
	case errors.Is(err, procurement.ErrAIRecommendationNotFound):
		writeProblem(w, r, http.StatusNotFound, "ai-recommendation-not-found", "Không tìm thấy khuyến nghị", "Khuyến nghị không tồn tại hoặc nằm ngoài phạm vi tổ chức.")
	case errors.Is(err, procurement.ErrAIRecommendationVersion):
		writeProblem(w, r, http.StatusConflict, "ai-recommendation-version-conflict", "Khuyến nghị đã thay đổi", "Hãy tải lại trung tâm khuyến nghị trước khi ra quyết định.")
	case errors.Is(err, procurement.ErrInvalidAIAction):
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid-ai-recommendation-action", "Thao tác khuyến nghị không hợp lệ", "Chỉ khuyến nghị đang chờ mới có thể được ra quyết định.")
	default:
		a.logger.Error(
			"procurement request failed",
			"error", err,
			"correlation_id", correlationIDFromContext(r.Context()),
		)
		writeProblem(w, r, http.StatusInternalServerError, "internal", "Lỗi máy chủ", "Không thể hoàn tất thao tác mua sắm.")
	}
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, destination any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return &requestBodyError{
			Status: http.StatusUnsupportedMediaType,
			Code:   "unsupported-media-type",
			Title:  "Loại nội dung không được hỗ trợ",
			Detail: "Content-Type phải là application/json.",
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(destination); err != nil {
		if errors.Is(err, io.EOF) {
			return invalidBody("Nội dung yêu cầu phải chứa một đối tượng JSON.")
		}
		return invalidBody("Nội dung yêu cầu phải là JSON hợp lệ và chỉ chứa các trường được hỗ trợ.")
	}
	if err = decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return invalidBody("Nội dung yêu cầu phải chứa chính xác một đối tượng JSON.")
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
		Title:  "Nội dung yêu cầu không hợp lệ",
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
			violations = append(violations, procurement.FieldViolation{Field: key, Message: "Tham số truy vấn này không được hỗ trợ."})
		}
		if len(values) != 1 {
			violations = append(violations, procurement.FieldViolation{Field: key, Message: "Chỉ được chỉ định tối đa một lần."})
		}
	}

	if value := query.Get("page"); value != "" {
		page, err := strconv.Atoi(value)
		if err != nil || page < 1 {
			violations = append(violations, procurement.FieldViolation{Field: "page", Message: "Phải là số nguyên lớn hơn hoặc bằng 1."})
		} else {
			input.Page = page
		}
	}
	if value := query.Get("pageSize"); value != "" {
		pageSize, err := strconv.Atoi(value)
		if err != nil || pageSize < 1 || pageSize > 100 {
			violations = append(violations, procurement.FieldViolation{Field: "pageSize", Message: "Phải là số nguyên trong khoảng từ 1 đến 100."})
		} else {
			input.PageSize = pageSize
		}
	}
	if value := query.Get("status"); value != "" {
		status, valid := procurement.ParseStatus(value)
		if !valid {
			violations = append(violations, procurement.FieldViolation{Field: "status", Message: "Phải là trạng thái phiếu mua sắm được hỗ trợ."})
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
			violations = append(violations, procurement.FieldViolation{Field: field, Message: "Không được vượt quá 100 ký tự."})
		}
	}
	input.From = strings.TrimSpace(query.Get("from"))
	input.To = strings.TrimSpace(query.Get("to"))
	for field, value := range map[string]string{"from": input.From, "to": input.To} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				violations = append(violations, procurement.FieldViolation{Field: field, Message: "Phải có định dạng YYYY-MM-DD."})
			}
		}
	}
	input.MinAmount = strings.TrimSpace(query.Get("minAmount"))
	input.MaxAmount = strings.TrimSpace(query.Get("maxAmount"))
	for field, value := range map[string]string{"minAmount": input.MinAmount, "maxAmount": input.MaxAmount} {
		if value != "" && !regexp.MustCompile(`^(0|[1-9][0-9]{0,14})(\.[0-9]{1,4})?$`).MatchString(value) {
			violations = append(violations, procurement.FieldViolation{Field: field, Message: "Phải là số tiền không âm."})
		}
	}
	if input.MinAmount != "" && input.MaxAmount != "" {
		minAmount, _ := new(big.Rat).SetString(input.MinAmount)
		maxAmount, _ := new(big.Rat).SetString(input.MaxAmount)
		if minAmount != nil && maxAmount != nil && minAmount.Cmp(maxAmount) > 0 {
			violations = append(violations, procurement.FieldViolation{Field: "minAmount", Message: "Không được lớn hơn maxAmount."})
		}
	}
	if value := strings.TrimSpace(query.Get("sort")); value != "" {
		if value != "createdAt" && value != "updatedAt" && value != "amount" && value != "code" {
			violations = append(violations, procurement.FieldViolation{Field: "sort", Message: "Phải là createdAt, updatedAt, amount hoặc code."})
		} else {
			input.Sort = value
		}
	}
	if value := strings.ToLower(strings.TrimSpace(query.Get("direction"))); value != "" {
		if value != "asc" && value != "desc" {
			violations = append(violations, procurement.FieldViolation{Field: "direction", Message: "Phải là asc (tăng dần) hoặc desc (giảm dần)."})
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
				Field: key, Message: "Tham số truy vấn này không được hỗ trợ.",
			})
		}
		if len(values) != 1 {
			violations = append(violations, procurement.FieldViolation{
				Field: key, Message: "Chỉ được chỉ định tối đa một lần.",
			})
		}
	}
	if value := query.Get("page"); value != "" {
		page, err := strconv.Atoi(value)
		if err != nil || page < 1 {
			violations = append(violations, procurement.FieldViolation{
				Field: "page", Message: "Phải là số nguyên lớn hơn hoặc bằng 1.",
			})
		} else {
			input.Page = page
		}
	}
	if value := query.Get("pageSize"); value != "" {
		pageSize, err := strconv.Atoi(value)
		if err != nil || pageSize < 1 || pageSize > 100 {
			violations = append(violations, procurement.FieldViolation{
				Field: "pageSize", Message: "Phải là số nguyên trong khoảng từ 1 đến 100.",
			})
		} else {
			input.PageSize = pageSize
		}
	}
	return input, violations
}
