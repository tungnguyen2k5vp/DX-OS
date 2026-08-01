package procurement

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
)

type Status string
type Action string

const (
	StatusDraft            Status = "DRAFT"
	StatusSubmitted        Status = "SUBMITTED"
	StatusManagerApproved  Status = "MANAGER_APPROVED"
	StatusChangesRequested Status = "CHANGES_REQUESTED"
	StatusApproved         Status = "APPROVED"
	StatusRejected         Status = "REJECTED"
	StatusCancelled        Status = "CANCELLED"

	ActionSubmit         Action = "SUBMIT"
	ActionResubmit       Action = "RESUBMIT"
	ActionCancel         Action = "CANCEL"
	ActionApprove        Action = "APPROVE"
	ActionReject         Action = "REJECT"
	ActionRequestChanges Action = "REQUEST_CHANGES"
)

var (
	ErrForbidden             = errors.New("purchase request access is forbidden")
	ErrNotFound              = errors.New("purchase request not found")
	ErrVersionConflict       = errors.New("purchase request version conflict")
	ErrInvalidTransition     = errors.New("purchase request transition is invalid")
	ErrIdempotencyConflict   = errors.New("idempotency key has already been used")
	ErrBudgetNotFound        = errors.New("budget allocation not found")
	ErrBudgetNotConfigured   = errors.New("budget is not configured")
	ErrInsufficientBudget    = errors.New("available budget is insufficient")
	ErrBudgetReservation     = errors.New("budget reservation is missing")
	ErrBudgetBelowUsage      = errors.New("budget allocation cannot be below reserved and committed amounts")
	ErrBudgetVersionConflict = errors.New("budget allocation version conflict")
	ErrAttachmentNotFound    = errors.New("purchase request attachment not found")
	ErrAttachmentRequired    = errors.New("a required purchase request attachment is missing")
	ErrDocumentStore         = errors.New("document store operation failed")

	quantityPattern    = regexp.MustCompile(`^(0|[1-9][0-9]{0,10})(\.[0-9]{1,4})?$`)
	unitPricePattern   = regexp.MustCompile(`^(0|[1-9][0-9]{0,14})(\.[0-9]{1,4})?$`)
	currencyPattern    = regexp.MustCompile(`^[A-Z]{3}$`)
	idempotencyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,254}$`)
	maxMoney, _        = new(big.Rat).SetString("999999999999999.9999")
	validStatuses      = []Status{
		StatusDraft,
		StatusSubmitted,
		StatusManagerApproved,
		StatusChangesRequested,
		StatusApproved,
		StatusRejected,
		StatusCancelled,
	}
)

const MaxAttachmentSize int64 = 10 * 1024 * 1024

var AllowedAttachmentContentTypes = []string{
	"application/pdf",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"image/jpeg",
	"image/png",
}

type FieldViolation struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationError struct {
	Violations []FieldViolation
}

func (e *ValidationError) Error() string {
	return "purchase request validation failed"
}

type CreateInput struct {
	Title      string
	Reason     string
	Currency   string
	CostCenter string
	Items      []CreateItemInput
}

type CreateItemInput struct {
	Description string
	Quantity    string
	Unit        string
	UnitPrice   string
}

type UpdateInput struct {
	CreateInput
	ExpectedVersion int64
}

type TransitionInput struct {
	Action          Action
	ExpectedVersion int64
	Comment         string
	IdempotencyKey  string
	CorrelationID   string
}

type ActorContext struct {
	UserID         string
	DepartmentID   string
	OrganizationID string
	Roles          []string
}

type RequestContext struct {
	RequesterID    string
	DepartmentID   string
	OrganizationID string
	Status         Status
}

type TransitionDecision struct {
	ToStatus  Status
	EventType string
}

type Item struct {
	ID          string `json:"id"`
	LineNumber  int    `json:"lineNumber"`
	Description string `json:"description"`
	Quantity    string `json:"quantity"`
	Unit        string `json:"unit"`
	UnitPrice   string `json:"unitPrice"`
	LineTotal   string `json:"lineTotal"`
}

type PurchaseRequest struct {
	ID             string    `json:"id"`
	RequestCode    string    `json:"requestCode"`
	RequesterID    string    `json:"requesterId"`
	RequesterName  string    `json:"requesterName"`
	DepartmentID   string    `json:"departmentId"`
	DepartmentName string    `json:"departmentName"`
	Title          string    `json:"title"`
	Reason         string    `json:"reason"`
	Currency       string    `json:"currency"`
	TotalAmount    string    `json:"totalAmount"`
	CostCenter     string    `json:"costCenter"`
	Status         Status    `json:"status"`
	Version        int64     `json:"version"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	Items          []Item    `json:"items,omitempty"`
}

type ListInput struct {
	Page     int
	PageSize int
	Status   *Status
}

type ListResult struct {
	Items    []PurchaseRequest `json:"items"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
	Total    int64             `json:"total"`
	Pages    int               `json:"pages"`
}

type TimelineInput struct {
	Page     int
	PageSize int
}

type TimelineEvent struct {
	ID            string    `json:"id"`
	EventType     string    `json:"eventType"`
	FromStatus    *Status   `json:"fromStatus"`
	ToStatus      Status    `json:"toStatus"`
	ActorName     string    `json:"actorName"`
	ActorRoles    []string  `json:"actorRoles"`
	Comment       *string   `json:"comment"`
	OccurredAt    time.Time `json:"occurredAt"`
	CorrelationID *string   `json:"correlationId"`
}

type TimelineResult struct {
	Items    []TimelineEvent `json:"items"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
	Total    int64           `json:"total"`
	Pages    int             `json:"pages"`
}

type BudgetSummaryInput struct {
	CostCenter string
	Currency   string
}

type BudgetSummary struct {
	PeriodCode      string `json:"periodCode"`
	PeriodStart     string `json:"periodStart"`
	PeriodEnd       string `json:"periodEnd"`
	CostCenter      string `json:"costCenter"`
	Currency        string `json:"currency"`
	AllocatedAmount string `json:"allocatedAmount"`
	ReservedAmount  string `json:"reservedAmount"`
	CommittedAmount string `json:"committedAmount"`
	AvailableAmount string `json:"availableAmount"`
}

type BudgetCheck struct {
	Configured       bool           `json:"configured"`
	Result           string         `json:"result"`
	RequestedAmount  string         `json:"requestedAmount"`
	ReservationState *string        `json:"reservationState"`
	Summary          *BudgetSummary `json:"summary"`
}

type BudgetAllocation struct {
	ID              string `json:"id"`
	PeriodCode      string `json:"periodCode"`
	PeriodStart     string `json:"periodStart"`
	PeriodEnd       string `json:"periodEnd"`
	CostCenter      string `json:"costCenter"`
	Currency        string `json:"currency"`
	AllocatedAmount string `json:"allocatedAmount"`
	ReservedAmount  string `json:"reservedAmount"`
	CommittedAmount string `json:"committedAmount"`
	AvailableAmount string `json:"availableAmount"`
	Utilization     string `json:"utilization"`
	AlertLevel      string `json:"alertLevel"`
	Version         int64  `json:"version"`
}

type BudgetCurrencyTotal struct {
	Currency        string `json:"currency"`
	AllocatedAmount string `json:"allocatedAmount"`
	ReservedAmount  string `json:"reservedAmount"`
	CommittedAmount string `json:"committedAmount"`
	AvailableAmount string `json:"availableAmount"`
}

type BudgetReservation struct {
	ID           string     `json:"id"`
	PurchaseID   string     `json:"purchaseRequestId"`
	RequestCode  string     `json:"requestCode"`
	RequestTitle string     `json:"requestTitle"`
	CostCenter   string     `json:"costCenter"`
	Currency     string     `json:"currency"`
	Amount       string     `json:"amount"`
	Status       string     `json:"status"`
	ReservedBy   string     `json:"reservedBy"`
	ReservedAt   time.Time  `json:"reservedAt"`
	CommittedAt  *time.Time `json:"committedAt"`
	ReleasedAt   *time.Time `json:"releasedAt"`
}

type BudgetAdjustment struct {
	ID             string    `json:"id"`
	AllocationID   string    `json:"allocationId"`
	CostCenter     string    `json:"costCenter"`
	Currency       string    `json:"currency"`
	PreviousAmount string    `json:"previousAmount"`
	AdjustedAmount string    `json:"adjustedAmount"`
	Reason         string    `json:"reason"`
	ActorName      string    `json:"actorName"`
	CreatedAt      time.Time `json:"createdAt"`
}

type BudgetDashboard struct {
	Allocations  []BudgetAllocation    `json:"allocations"`
	Totals       []BudgetCurrencyTotal `json:"totals"`
	Reservations []BudgetReservation   `json:"reservations"`
	Adjustments  []BudgetAdjustment    `json:"adjustments"`
	AlertCount   int                   `json:"alertCount"`
	CanManage    bool                  `json:"canManage"`
}

type AdjustBudgetInput struct {
	AllocatedAmount string
	ExpectedVersion int64
	Reason          string
	IdempotencyKey  string
	CorrelationID   string
}

type DocumentType string

const (
	DocumentTypeQuotation     DocumentType = "QUOTATION"
	DocumentTypeSpecification DocumentType = "SPECIFICATION"
	DocumentTypeContract      DocumentType = "CONTRACT"
	DocumentTypeOther         DocumentType = "OTHER"
)

type UploadAttachmentInput struct {
	DocumentType  DocumentType
	FileName      string
	ContentType   string
	Content       []byte
	CorrelationID string
}

type Attachment struct {
	ID             string       `json:"id"`
	PurchaseID     string       `json:"purchaseRequestId"`
	DocumentType   DocumentType `json:"documentType"`
	FileName       string       `json:"fileName"`
	ContentType    string       `json:"contentType"`
	SizeBytes      int64        `json:"sizeBytes"`
	ChecksumSHA256 string       `json:"checksumSha256"`
	UploadedBy     string       `json:"uploadedBy"`
	UploadedByName string       `json:"uploadedByName"`
	UploadedAt     time.Time    `json:"uploadedAt"`
}

type AttachmentList struct {
	Items                []Attachment `json:"items"`
	Required             bool         `json:"required"`
	RequirementMet       bool         `json:"requirementMet"`
	RequiredDocumentType DocumentType `json:"requiredDocumentType,omitempty"`
	ThresholdAmount      string       `json:"thresholdAmount,omitempty"`
	MaxSizeBytes         int64        `json:"maxSizeBytes"`
	AllowedContentTypes  []string     `json:"allowedContentTypes"`
}

type AttachmentContent struct {
	Attachment Attachment
	Content    []byte
}

func ValidateAttachment(input *UploadAttachmentInput) error {
	var violations []FieldViolation
	switch input.DocumentType {
	case DocumentTypeQuotation, DocumentTypeSpecification, DocumentTypeContract, DocumentTypeOther:
	default:
		violations = append(violations, FieldViolation{
			Field: "documentType", Message: "Loại tài liệu không hợp lệ.",
		})
	}
	input.FileName = strings.TrimSpace(input.FileName)
	if input.FileName == "" || len([]rune(input.FileName)) > 255 ||
		strings.ContainsAny(input.FileName, `/\`) ||
		strings.ContainsAny(input.FileName, "\x00\r\n") {
		violations = append(violations, FieldViolation{
			Field: "file", Message: "Tên tệp không hợp lệ hoặc dài quá 255 ký tự.",
		})
	}
	if !slices.Contains(AllowedAttachmentContentTypes, input.ContentType) {
		violations = append(violations, FieldViolation{
			Field: "file", Message: "Chỉ hỗ trợ PDF, DOCX, XLSX, JPG và PNG.",
		})
	}
	if len(input.Content) == 0 || int64(len(input.Content)) > MaxAttachmentSize {
		violations = append(violations, FieldViolation{
			Field: "file", Message: "Tệp phải có dung lượng từ 1 byte đến 10 MB.",
		})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

type ScopeKind int

const (
	ScopeOwn ScopeKind = iota
	ScopeDepartment
	ScopeFinance
	ScopeAll
)

func ScopeFor(principal auth.Principal) (ScopeKind, error) {
	switch {
	case hasRole(principal.Roles, "auditor"):
		return ScopeAll, nil
	case hasRole(principal.Roles, "finance"):
		return ScopeFinance, nil
	case hasRole(principal.Roles, "department_manager"):
		return ScopeDepartment, nil
	case hasRole(principal.Roles, "employee"):
		return ScopeOwn, nil
	default:
		return ScopeOwn, ErrForbidden
	}
}

func CanCreate(principal auth.Principal) bool {
	return hasRole(principal.Roles, "employee") || hasRole(principal.Roles, "department_manager")
}

func ParseStatus(value string) (Status, bool) {
	status := Status(strings.ToUpper(strings.TrimSpace(value)))
	return status, slices.Contains(validStatuses, status)
}

func ValidateCreate(input *CreateInput) error {
	input.Title = strings.TrimSpace(input.Title)
	input.Reason = strings.TrimSpace(input.Reason)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.CostCenter = strings.TrimSpace(input.CostCenter)

	var violations []FieldViolation
	if length := len([]rune(input.Title)); length < 3 || length > 255 {
		violations = append(violations, FieldViolation{Field: "title", Message: "must contain between 3 and 255 characters"})
	}
	if length := len([]rune(input.Reason)); length < 10 || length > 5000 {
		violations = append(violations, FieldViolation{Field: "reason", Message: "must contain between 10 and 5000 characters"})
	}
	if !currencyPattern.MatchString(input.Currency) {
		violations = append(violations, FieldViolation{Field: "currency", Message: "must be a three-letter uppercase currency code"})
	}
	if length := len([]rune(input.CostCenter)); length < 1 || length > 100 {
		violations = append(violations, FieldViolation{Field: "costCenter", Message: "must contain between 1 and 100 characters"})
	}
	if len(input.Items) < 1 || len(input.Items) > 100 {
		violations = append(violations, FieldViolation{Field: "items", Message: "must contain between 1 and 100 items"})
	}

	total := new(big.Rat)
	totalCanBeCalculated := true
	for index := range input.Items {
		item := &input.Items[index]
		item.Description = strings.TrimSpace(item.Description)
		item.Quantity = strings.TrimSpace(item.Quantity)
		item.Unit = strings.TrimSpace(item.Unit)
		item.UnitPrice = strings.TrimSpace(item.UnitPrice)
		prefix := fmt.Sprintf("items[%d].", index)

		if length := len([]rune(item.Description)); length < 2 || length > 500 {
			violations = append(violations, FieldViolation{Field: prefix + "description", Message: "must contain between 2 and 500 characters"})
		}
		quantity, quantityValid := decimal(item.Quantity, quantityPattern, true)
		if !quantityValid {
			violations = append(violations, FieldViolation{Field: prefix + "quantity", Message: "must be a positive decimal with at most 4 fractional digits"})
		}
		if length := len([]rune(item.Unit)); length < 1 || length > 50 {
			violations = append(violations, FieldViolation{Field: prefix + "unit", Message: "must contain between 1 and 50 characters"})
		}
		unitPrice, unitPriceValid := decimal(item.UnitPrice, unitPricePattern, false)
		if !unitPriceValid {
			violations = append(violations, FieldViolation{Field: prefix + "unitPrice", Message: "must be a non-negative decimal with at most 4 fractional digits"})
		}
		if quantityValid && unitPriceValid {
			lineTotal := new(big.Rat).Mul(quantity, unitPrice)
			if lineTotal.Cmp(maxMoney) > 0 {
				violations = append(violations, FieldViolation{Field: prefix + "unitPrice", Message: "produces a line total above the supported monetary limit"})
				totalCanBeCalculated = false
			} else {
				total.Add(total, lineTotal)
			}
		} else {
			totalCanBeCalculated = false
		}
	}
	if totalCanBeCalculated && total.Cmp(maxMoney) > 0 {
		violations = append(violations, FieldViolation{Field: "items", Message: "combined total exceeds the supported monetary limit"})
	}

	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func ValidateUpdate(input *UpdateInput) error {
	var violations []FieldViolation
	if input.ExpectedVersion < 1 {
		violations = append(violations, FieldViolation{
			Field: "expectedVersion", Message: "must be an integer greater than or equal to 1",
		})
	}
	if err := ValidateCreate(&input.CreateInput); err != nil {
		var validationError *ValidationError
		if errors.As(err, &validationError) {
			violations = append(violations, validationError.Violations...)
		} else {
			return err
		}
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func ValidateBudgetSummaryInput(input *BudgetSummaryInput) error {
	input.CostCenter = strings.TrimSpace(input.CostCenter)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))

	var violations []FieldViolation
	if length := len([]rune(input.CostCenter)); length < 1 || length > 100 {
		violations = append(violations, FieldViolation{
			Field: "costCenter", Message: "must contain between 1 and 100 characters",
		})
	}
	if !currencyPattern.MatchString(input.Currency) {
		violations = append(violations, FieldViolation{
			Field: "currency", Message: "must be a three-letter uppercase currency code",
		})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func ValidateAdjustBudgetInput(input *AdjustBudgetInput) error {
	input.AllocatedAmount = strings.TrimSpace(input.AllocatedAmount)
	input.Reason = strings.TrimSpace(input.Reason)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)

	var violations []FieldViolation
	if _, valid := decimal(input.AllocatedAmount, unitPricePattern, false); !valid {
		violations = append(violations, FieldViolation{
			Field:   "allocatedAmount",
			Message: "must be a non-negative decimal with at most 4 fractional digits",
		})
	}
	if input.ExpectedVersion < 1 {
		violations = append(violations, FieldViolation{
			Field: "expectedVersion", Message: "must be an integer greater than or equal to 1",
		})
	}
	if length := len([]rune(input.Reason)); length < 10 || length > 1000 {
		violations = append(violations, FieldViolation{
			Field: "reason", Message: "must contain between 10 and 1000 characters",
		})
	}
	if !idempotencyPattern.MatchString(input.IdempotencyKey) {
		violations = append(violations, FieldViolation{
			Field: "Idempotency-Key", Message: "must contain 8 to 255 safe ASCII characters",
		})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func ParseAction(value string) (Action, bool) {
	action := Action(strings.ToUpper(strings.TrimSpace(value)))
	switch action {
	case ActionSubmit, ActionResubmit, ActionCancel, ActionApprove, ActionReject, ActionRequestChanges:
		return action, true
	default:
		return "", false
	}
}

func ValidateTransition(input *TransitionInput) error {
	input.Comment = strings.TrimSpace(input.Comment)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)

	var violations []FieldViolation
	if action, valid := ParseAction(string(input.Action)); !valid {
		violations = append(violations, FieldViolation{
			Field: "action", Message: "must be a supported purchase request action",
		})
	} else {
		input.Action = action
	}
	if input.ExpectedVersion < 1 {
		violations = append(violations, FieldViolation{
			Field: "expectedVersion", Message: "must be an integer greater than or equal to 1",
		})
	}
	if len([]rune(input.Comment)) > 2000 {
		violations = append(violations, FieldViolation{
			Field: "comment", Message: "must contain at most 2000 characters",
		})
	}
	if (input.Action == ActionReject || input.Action == ActionRequestChanges) && input.Comment == "" {
		violations = append(violations, FieldViolation{
			Field: "comment", Message: "is required for REJECT and REQUEST_CHANGES",
		})
	}
	if !idempotencyPattern.MatchString(input.IdempotencyKey) {
		violations = append(violations, FieldViolation{
			Field: "Idempotency-Key", Message: "must contain 8 to 255 safe ASCII characters",
		})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func DecideTransition(
	actor ActorContext,
	request RequestContext,
	action Action,
) (TransitionDecision, error) {
	if actor.UserID == "" || request.RequesterID == "" {
		return TransitionDecision{}, ErrForbidden
	}

	switch request.Status {
	case StatusDraft:
		if actor.UserID != request.RequesterID {
			return TransitionDecision{}, ErrNotFound
		}
		switch action {
		case ActionSubmit:
			return TransitionDecision{ToStatus: StatusSubmitted, EventType: "SUBMITTED"}, nil
		case ActionCancel:
			return TransitionDecision{ToStatus: StatusCancelled, EventType: "CANCELLED"}, nil
		}
	case StatusChangesRequested:
		if actor.UserID != request.RequesterID {
			return TransitionDecision{}, ErrNotFound
		}
		switch action {
		case ActionResubmit:
			return TransitionDecision{ToStatus: StatusSubmitted, EventType: "RESUBMITTED"}, nil
		case ActionCancel:
			return TransitionDecision{ToStatus: StatusCancelled, EventType: "CANCELLED"}, nil
		}
	case StatusSubmitted:
		if !hasRole(actor.Roles, "department_manager") {
			return TransitionDecision{}, ErrForbidden
		}
		if actor.DepartmentID != request.DepartmentID {
			return TransitionDecision{}, ErrNotFound
		}
		if actor.UserID == request.RequesterID {
			return TransitionDecision{}, ErrForbidden
		}
		switch action {
		case ActionApprove:
			return TransitionDecision{ToStatus: StatusManagerApproved, EventType: "MANAGER_APPROVED"}, nil
		case ActionReject:
			return TransitionDecision{ToStatus: StatusRejected, EventType: "REJECTED"}, nil
		case ActionRequestChanges:
			return TransitionDecision{ToStatus: StatusChangesRequested, EventType: "CHANGES_REQUESTED"}, nil
		}
	case StatusManagerApproved:
		if !hasRole(actor.Roles, "finance") {
			return TransitionDecision{}, ErrForbidden
		}
		if actor.OrganizationID != request.OrganizationID {
			return TransitionDecision{}, ErrNotFound
		}
		if actor.UserID == request.RequesterID {
			return TransitionDecision{}, ErrForbidden
		}
		switch action {
		case ActionApprove:
			return TransitionDecision{ToStatus: StatusApproved, EventType: "FINANCE_APPROVED"}, nil
		case ActionReject:
			return TransitionDecision{ToStatus: StatusRejected, EventType: "REJECTED"}, nil
		case ActionRequestChanges:
			return TransitionDecision{ToStatus: StatusChangesRequested, EventType: "CHANGES_REQUESTED"}, nil
		}
	}
	return TransitionDecision{}, ErrInvalidTransition
}

func decimal(value string, pattern *regexp.Regexp, strictlyPositive bool) (*big.Rat, bool) {
	if !pattern.MatchString(value) {
		return nil, false
	}
	number, ok := new(big.Rat).SetString(value)
	if !ok {
		return nil, false
	}
	if strictlyPositive {
		return number, number.Sign() > 0
	}
	return number, number.Sign() >= 0
}

func hasRole(roles []string, expected string) bool {
	return slices.Contains(roles, expected)
}
