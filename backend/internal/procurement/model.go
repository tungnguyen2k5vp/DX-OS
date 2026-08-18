package procurement

import (
	"archive/zip"
	"bytes"
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
	ErrForbidden                = errors.New("purchase request access is forbidden")
	ErrNotFound                 = errors.New("purchase request not found")
	ErrVersionConflict          = errors.New("purchase request version conflict")
	ErrInvalidTransition        = errors.New("purchase request transition is invalid")
	ErrIdempotencyConflict      = errors.New("idempotency key has already been used")
	ErrBudgetNotFound           = errors.New("budget allocation not found")
	ErrBudgetNotConfigured      = errors.New("budget is not configured")
	ErrInsufficientBudget       = errors.New("available budget is insufficient")
	ErrBudgetReservation        = errors.New("budget reservation is missing")
	ErrBudgetBelowUsage         = errors.New("budget allocation cannot be below reserved and committed amounts")
	ErrBudgetVersionConflict    = errors.New("budget allocation version conflict")
	ErrAttachmentNotFound       = errors.New("purchase request attachment not found")
	ErrAttachmentRequired       = errors.New("a required purchase request attachment is missing")
	ErrDocumentStore            = errors.New("document store operation failed")
	ErrSupplierNotFound         = errors.New("supplier not found")
	ErrSupplierConflict         = errors.New("supplier code or tax code already exists")
	ErrSupplierVersion          = errors.New("supplier version conflict")
	ErrPurchaseOrderNotFound    = errors.New("purchase order not found")
	ErrPurchaseOrderConflict    = errors.New("purchase order already exists or idempotency key conflicts")
	ErrInvalidFulfillment       = errors.New("purchase order operation is invalid")
	ErrInvoiceNotFound          = errors.New("purchase invoice not found")
	ErrInvoiceConflict          = errors.New("purchase invoice already exists or conflicts")
	ErrInvoiceVersion           = errors.New("purchase invoice version conflict")
	ErrInvalidInvoiceAction     = errors.New("purchase invoice action is invalid")
	ErrInvoiceMismatch          = errors.New("purchase invoice does not match the order and receipt")
	ErrPolicyNotFound           = errors.New("operating policy not found")
	ErrPolicyVersion            = errors.New("operating policy version conflict")
	ErrAuditCaseNotFound        = errors.New("audit case not found")
	ErrAuditCaseVersion         = errors.New("audit case version conflict")
	ErrAIRecommendationNotFound = errors.New("AI recommendation not found")
	ErrAIRecommendationVersion  = errors.New("AI recommendation version conflict")
	ErrInvalidAIAction          = errors.New("AI recommendation action is invalid")
	ErrAdminUserNotFound        = errors.New("admin user not found")
	ErrAdminDepartmentNotFound  = errors.New("admin department not found")
	ErrAdminVersion             = errors.New("admin resource version conflict")
	ErrAdminConflict            = errors.New("admin resource conflict")

	quantityPattern      = regexp.MustCompile(`^(0|[1-9][0-9]{0,10})(\.[0-9]{1,4})?$`)
	unitPricePattern     = regexp.MustCompile(`^(0|[1-9][0-9]{0,14})(\.[0-9]{1,4})?$`)
	currencyPattern      = regexp.MustCompile(`^[A-Z]{3}$`)
	idempotencyPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,254}$`)
	supplierCodePattern  = regexp.MustCompile(`^[A-Z0-9][A-Z0-9._-]{1,49}$`)
	emailPattern         = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	invoiceNumberPattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9./_-]{1,99}$`)
	uuidPatternForDomain = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	maxMoney, _          = new(big.Rat).SetString("999999999999999.9999")
	validStatuses        = []Status{
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
	Page       int
	PageSize   int
	Status     *Status
	Search     string
	Department string
	CostCenter string
	Requester  string
	From       string
	To         string
	MinAmount  string
	MaxAmount  string
	Sort       string
	Direction  string
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

type CommentInput struct {
	Body          string
	CorrelationID string
}

type Comment struct {
	ID          string    `json:"id"`
	Body        string    `json:"body"`
	AuthorID    string    `json:"authorId"`
	AuthorName  string    `json:"authorName"`
	AuthorRoles []string  `json:"authorRoles"`
	CreatedAt   time.Time `json:"createdAt"`
}

type CommentList struct {
	Items []Comment `json:"items"`
	Total int64     `json:"total"`
}

type WorkTask struct {
	PurchaseRequestID string     `json:"purchaseRequestId"`
	RequestCode       string     `json:"requestCode"`
	Title             string     `json:"title"`
	RequesterName     string     `json:"requesterName"`
	DepartmentName    string     `json:"departmentName"`
	Status            Status     `json:"status"`
	TaskType          string     `json:"taskType"`
	Currency          string     `json:"currency"`
	TotalAmount       string     `json:"totalAmount"`
	DueAt             *time.Time `json:"dueAt"`
	Overdue           bool       `json:"overdue"`
	Urgency           string     `json:"urgency"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

type WorkSummary struct {
	Items        []WorkTask `json:"items"`
	Total        int        `json:"total"`
	OverdueCount int        `json:"overdueCount"`
	DueSoonCount int        `json:"dueSoonCount"`
}

type Supplier struct {
	ID                string    `json:"id"`
	Code              string    `json:"code"`
	Name              string    `json:"name"`
	TaxCode           string    `json:"taxCode,omitempty"`
	ContactName       string    `json:"contactName,omitempty"`
	Email             string    `json:"email,omitempty"`
	Phone             string    `json:"phone,omitempty"`
	Address           string    `json:"address,omitempty"`
	BankName          string    `json:"bankName,omitempty"`
	BankAccountNumber string    `json:"bankAccountNumber,omitempty"`
	ContractReference string    `json:"contractReference,omitempty"`
	ContractExpiresOn string    `json:"contractExpiresOn,omitempty"`
	ComplianceStatus  string    `json:"complianceStatus"`
	PerformanceScore  string    `json:"performanceScore,omitempty"`
	BusinessNote      string    `json:"businessNote,omitempty"`
	Status            string    `json:"status"`
	RiskLevel         string    `json:"riskLevel"`
	Version           int64     `json:"version"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type SupplierInput struct {
	Code              string
	Name              string
	TaxCode           string
	ContactName       string
	Email             string
	Phone             string
	Address           string
	BankName          string
	BankAccountNumber string
	ContractReference string
	ContractExpiresOn string
	ComplianceStatus  string
	PerformanceScore  string
	BusinessNote      string
	Status            string
	RiskLevel         string
}

type UpdateSupplierInput struct {
	SupplierInput
	ExpectedVersion int64
}

type SupplierList struct {
	Items     []Supplier `json:"items"`
	Total     int        `json:"total"`
	CanManage bool       `json:"canManage"`
}

type PurchaseOrder struct {
	ID                 string     `json:"id"`
	PurchaseRequestID  string     `json:"purchaseRequestId"`
	RequestCode        string     `json:"requestCode"`
	RequestTitle       string     `json:"requestTitle"`
	RequesterName      string     `json:"requesterName"`
	DepartmentName     string     `json:"departmentName"`
	Currency           string     `json:"currency"`
	TotalAmount        string     `json:"totalAmount"`
	OrderCode          *string    `json:"orderCode"`
	SupplierID         *string    `json:"supplierId"`
	SupplierCode       *string    `json:"supplierCode"`
	SupplierName       *string    `json:"supplierName"`
	ExternalReference  *string    `json:"externalReference"`
	ExpectedDeliveryOn *string    `json:"expectedDeliveryOn"`
	ActualDeliveryOn   *string    `json:"actualDeliveryOn"`
	Status             string     `json:"status"`
	Note               *string    `json:"note"`
	Version            int64      `json:"version"`
	OrderedAt          *time.Time `json:"orderedAt"`
	ReceivedAt         *time.Time `json:"receivedAt"`
	CancelledAt        *time.Time `json:"cancelledAt"`
	CancellationReason *string    `json:"cancellationReason"`
	ReceiptCount       int        `json:"receiptCount"`
	DeliveryOverdue    bool       `json:"deliveryOverdue"`
	CanPlaceOrder      bool       `json:"canPlaceOrder"`
	CanConfirmReceipt  bool       `json:"canConfirmReceipt"`
	CanManageOrder     bool       `json:"canManageOrder"`
}

type OperationsBoard struct {
	Items                []PurchaseOrder `json:"items"`
	Total                int             `json:"total"`
	AwaitingOrderCount   int             `json:"awaitingOrderCount"`
	InDeliveryCount      int             `json:"inDeliveryCount"`
	OverdueDeliveryCount int             `json:"overdueDeliveryCount"`
	ReceivedCount        int             `json:"receivedCount"`
	PartialCount         int             `json:"partialCount"`
	ExceptionCount       int             `json:"exceptionCount"`
	CancelledCount       int             `json:"cancelledCount"`
}

type CreatePurchaseOrderInput struct {
	PurchaseRequestID  string
	SupplierID         string
	ExternalReference  string
	ExpectedDeliveryOn string
	Note               string
	IdempotencyKey     string
	CorrelationID      string
}

type ConfirmReceiptInput struct {
	ExpectedVersion  int64
	ActualDeliveryOn string
	CorrelationID    string
}

type InvoiceBoardItem struct {
	PurchaseOrderID   string     `json:"purchaseOrderId"`
	PurchaseRequestID string     `json:"purchaseRequestId"`
	RequestCode       string     `json:"requestCode"`
	RequestTitle      string     `json:"requestTitle"`
	RequesterName     string     `json:"requesterName"`
	DepartmentName    string     `json:"departmentName"`
	SupplierID        string     `json:"supplierId"`
	SupplierCode      string     `json:"supplierCode"`
	SupplierName      string     `json:"supplierName"`
	OrderCode         string     `json:"orderCode"`
	OrderStatus       string     `json:"orderStatus"`
	OrderAmount       string     `json:"orderAmount"`
	OrderCurrency     string     `json:"orderCurrency"`
	ActualDeliveryOn  *string    `json:"actualDeliveryOn"`
	InvoiceID         *string    `json:"invoiceId"`
	InvoiceNumber     *string    `json:"invoiceNumber"`
	IssuedOn          *string    `json:"issuedOn"`
	DueOn             *string    `json:"dueOn"`
	InvoiceAmount     *string    `json:"invoiceAmount"`
	InvoiceCurrency   *string    `json:"invoiceCurrency"`
	InvoiceStatus     *string    `json:"invoiceStatus"`
	MatchStatus       string     `json:"matchStatus"`
	Note              *string    `json:"note"`
	Version           int64      `json:"version"`
	PaymentReference  *string    `json:"paymentReference"`
	PaidOn            *string    `json:"paidOn"`
	PaidAmount        string     `json:"paidAmount"`
	RemainingAmount   string     `json:"remainingAmount"`
	PaymentCount      int        `json:"paymentCount"`
	InvoiceCreatedAt  *time.Time `json:"invoiceCreatedAt"`
	InvoiceUpdatedAt  *time.Time `json:"invoiceUpdatedAt"`
	PaymentOverdue    bool       `json:"paymentOverdue"`
	CanManage         bool       `json:"canManage"`
}

type InvoiceBoard struct {
	Items                []InvoiceBoardItem `json:"items"`
	Total                int                `json:"total"`
	AwaitingInvoiceCount int                `json:"awaitingInvoiceCount"`
	NeedsReviewCount     int                `json:"needsReviewCount"`
	ReadyToPayCount      int                `json:"readyToPayCount"`
	OverdueCount         int                `json:"overdueCount"`
	PaidCount            int                `json:"paidCount"`
	CanManage            bool               `json:"canManage"`
}

type InvoiceInput struct {
	PurchaseOrderID string
	InvoiceNumber   string
	IssuedOn        string
	DueOn           string
	Amount          string
	Currency        string
	Note            string
	IdempotencyKey  string
	CorrelationID   string
}

type UpdateInvoiceInput struct {
	InvoiceNumber   string
	IssuedOn        string
	DueOn           string
	Amount          string
	Currency        string
	Note            string
	ExpectedVersion int64
	CorrelationID   string
}

type InvoiceActionInput struct {
	Action           string
	ExpectedVersion  int64
	Comment          string
	PaymentReference string
	PaidOn           string
	IdempotencyKey   string
	CorrelationID    string
}

type SLAPolicy struct {
	ProcessName string `json:"processName"`
	TargetHours int    `json:"targetHours"`
	Active      bool   `json:"active"`
	Version     int64  `json:"version"`
}

type AttachmentPolicy struct {
	ID                   string `json:"id"`
	Currency             string `json:"currency"`
	ThresholdAmount      string `json:"thresholdAmount"`
	RequiredDocumentType string `json:"requiredDocumentType"`
	Active               bool   `json:"active"`
	Version              int64  `json:"version"`
}

type PolicyCenter struct {
	SLA             []SLAPolicy        `json:"slaPolicies"`
	AttachmentRules []AttachmentPolicy `json:"attachmentRules"`
	CanManage       bool               `json:"canManage"`
}

type UpdateSLAPolicyInput struct {
	TargetHours     int
	Active          bool
	ExpectedVersion int64
	CorrelationID   string
}

type UpdateAttachmentPolicyInput struct {
	ThresholdAmount      string
	RequiredDocumentType string
	Active               bool
	ExpectedVersion      int64
	CorrelationID        string
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
	contentTypeAllowed := slices.Contains(AllowedAttachmentContentTypes, input.ContentType)
	if !contentTypeAllowed {
		violations = append(violations, FieldViolation{
			Field: "file", Message: "Chỉ hỗ trợ PDF, DOCX, XLSX, JPG và PNG.",
		})
	}
	contentSizeValid := len(input.Content) > 0 && int64(len(input.Content)) <= MaxAttachmentSize
	if !contentSizeValid {
		violations = append(violations, FieldViolation{
			Field: "file", Message: "Tệp phải có dung lượng từ 1 byte đến 10 MB.",
		})
	}
	if contentTypeAllowed && contentSizeValid &&
		(!attachmentNameMatchesType(input.FileName, input.ContentType) ||
			!attachmentContentMatchesType(input.Content, input.ContentType)) {
		violations = append(violations, FieldViolation{
			Field: "file", Message: "Phần mở rộng hoặc nội dung tệp không khớp với loại tệp đã khai báo.",
		})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func attachmentNameMatchesType(fileName, contentType string) bool {
	extensions := map[string][]string{
		"application/pdf": {".pdf"},
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": {".docx"},
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":       {".xlsx"},
		"image/jpeg": {".jpg", ".jpeg"},
		"image/png":  {".png"},
	}
	lowerName := strings.ToLower(fileName)
	for _, extension := range extensions[contentType] {
		if strings.HasSuffix(lowerName, extension) {
			return true
		}
	}
	return false
}

func attachmentContentMatchesType(content []byte, contentType string) bool {
	switch contentType {
	case "application/pdf":
		return bytes.HasPrefix(content, []byte("%PDF-"))
	case "image/jpeg":
		return len(content) >= 3 && content[0] == 0xff && content[1] == 0xd8 && content[2] == 0xff
	case "image/png":
		return bytes.HasPrefix(content, []byte("\x89PNG\r\n\x1a\n"))
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return officeArchiveContains(content, "word/document.xml")
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return officeArchiveContains(content, "xl/workbook.xml")
	default:
		return false
	}
}

func officeArchiveContains(content []byte, requiredEntry string) bool {
	archive, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return false
	}
	foundContentTypes := false
	foundRequiredEntry := false
	for _, file := range archive.File {
		switch file.Name {
		case "[Content_Types].xml":
			foundContentTypes = true
		case requiredEntry:
			foundRequiredEntry = true
		}
	}
	return foundContentTypes && foundRequiredEntry
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
	return !hasRole(principal.Roles, "auditor") &&
		(hasRole(principal.Roles, "employee") || hasRole(principal.Roles, "department_manager"))
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
		violations = append(violations, FieldViolation{Field: "title", Message: "Phải có từ 3 đến 255 ký tự."})
	}
	if length := len([]rune(input.Reason)); length < 10 || length > 5000 {
		violations = append(violations, FieldViolation{Field: "reason", Message: "Phải có từ 10 đến 5.000 ký tự."})
	}
	if !currencyPattern.MatchString(input.Currency) {
		violations = append(violations, FieldViolation{Field: "currency", Message: "Phải là mã tiền tệ gồm 3 chữ cái viết hoa."})
	}
	if length := len([]rune(input.CostCenter)); length < 1 || length > 100 {
		violations = append(violations, FieldViolation{Field: "costCenter", Message: "Phải có từ 1 đến 100 ký tự."})
	}
	if len(input.Items) < 1 || len(input.Items) > 100 {
		violations = append(violations, FieldViolation{Field: "items", Message: "Phải có từ 1 đến 100 dòng hàng."})
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
			violations = append(violations, FieldViolation{Field: prefix + "description", Message: "Phải có từ 2 đến 500 ký tự."})
		}
		quantity, quantityValid := decimal(item.Quantity, quantityPattern, true)
		if !quantityValid {
			violations = append(violations, FieldViolation{Field: prefix + "quantity", Message: "Phải là số dương và có tối đa 4 chữ số thập phân."})
		}
		if length := len([]rune(item.Unit)); length < 1 || length > 50 {
			violations = append(violations, FieldViolation{Field: prefix + "unit", Message: "Phải có từ 1 đến 50 ký tự."})
		}
		unitPrice, unitPriceValid := decimal(item.UnitPrice, unitPricePattern, false)
		if !unitPriceValid {
			violations = append(violations, FieldViolation{Field: prefix + "unitPrice", Message: "Phải là số không âm và có tối đa 4 chữ số thập phân."})
		}
		if quantityValid && unitPriceValid {
			lineTotal := new(big.Rat).Mul(quantity, unitPrice)
			if lineTotal.Cmp(maxMoney) > 0 {
				violations = append(violations, FieldViolation{Field: prefix + "unitPrice", Message: "Làm cho thành tiền vượt quá giới hạn hệ thống hỗ trợ."})
				totalCanBeCalculated = false
			} else {
				total.Add(total, lineTotal)
			}
		} else {
			totalCanBeCalculated = false
		}
	}
	if totalCanBeCalculated && total.Cmp(maxMoney) > 0 {
		violations = append(violations, FieldViolation{Field: "items", Message: "Tổng giá trị vượt quá giới hạn hệ thống hỗ trợ."})
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
			Field: "expectedVersion", Message: "Phải là số nguyên lớn hơn hoặc bằng 1.",
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
			Field: "costCenter", Message: "Phải có từ 1 đến 100 ký tự.",
		})
	}
	if !currencyPattern.MatchString(input.Currency) {
		violations = append(violations, FieldViolation{
			Field: "currency", Message: "Phải là mã tiền tệ gồm 3 chữ cái viết hoa.",
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
			Message: "Phải là số không âm và có tối đa 4 chữ số thập phân.",
		})
	}
	if input.ExpectedVersion < 1 {
		violations = append(violations, FieldViolation{
			Field: "expectedVersion", Message: "Phải là số nguyên lớn hơn hoặc bằng 1.",
		})
	}
	if length := len([]rune(input.Reason)); length < 10 || length > 1000 {
		violations = append(violations, FieldViolation{
			Field: "reason", Message: "Phải có từ 10 đến 1.000 ký tự.",
		})
	}
	if !idempotencyPattern.MatchString(input.IdempotencyKey) {
		violations = append(violations, FieldViolation{
			Field: "Idempotency-Key", Message: "Phải có từ 8 đến 255 ký tự ASCII an toàn.",
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
			Field: "action", Message: "Phải là thao tác phiếu mua sắm được hỗ trợ.",
		})
	} else {
		input.Action = action
	}
	if input.ExpectedVersion < 1 {
		violations = append(violations, FieldViolation{
			Field: "expectedVersion", Message: "Phải là số nguyên lớn hơn hoặc bằng 1.",
		})
	}
	if len([]rune(input.Comment)) > 2000 {
		violations = append(violations, FieldViolation{
			Field: "comment", Message: "Không được vượt quá 2.000 ký tự.",
		})
	}
	if (input.Action == ActionReject || input.Action == ActionRequestChanges) && input.Comment == "" {
		violations = append(violations, FieldViolation{
			Field: "comment", Message: "Bắt buộc khi từ chối hoặc yêu cầu chỉnh sửa.",
		})
	}
	if !idempotencyPattern.MatchString(input.IdempotencyKey) {
		violations = append(violations, FieldViolation{
			Field: "Idempotency-Key", Message: "Phải có từ 8 đến 255 ký tự ASCII an toàn.",
		})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func ValidateComment(input *CommentInput) error {
	input.Body = strings.TrimSpace(input.Body)
	var violations []FieldViolation
	if length := len([]rune(input.Body)); length < 1 || length > 2000 {
		violations = append(violations, FieldViolation{
			Field: "body", Message: "Phải có từ 1 đến 2.000 ký tự.",
		})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func ValidateSupplierInput(input *SupplierInput) error {
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.TaxCode = strings.TrimSpace(input.TaxCode)
	input.ContactName = strings.TrimSpace(input.ContactName)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Phone = strings.TrimSpace(input.Phone)
	input.Status = strings.ToUpper(strings.TrimSpace(input.Status))
	input.RiskLevel = strings.ToUpper(strings.TrimSpace(input.RiskLevel))
	input.Address = strings.TrimSpace(input.Address)
	input.BankName = strings.TrimSpace(input.BankName)
	input.BankAccountNumber = strings.TrimSpace(input.BankAccountNumber)
	input.ContractReference = strings.TrimSpace(input.ContractReference)
	input.ContractExpiresOn = strings.TrimSpace(input.ContractExpiresOn)
	input.ComplianceStatus = strings.ToUpper(strings.TrimSpace(input.ComplianceStatus))
	input.PerformanceScore = strings.TrimSpace(input.PerformanceScore)
	input.BusinessNote = strings.TrimSpace(input.BusinessNote)
	if input.ComplianceStatus == "" {
		input.ComplianceStatus = "PENDING"
	}
	var violations []FieldViolation
	if !supplierCodePattern.MatchString(input.Code) {
		violations = append(violations, FieldViolation{Field: "code", Message: "Phải có từ 2 đến 50 chữ cái viết hoa, chữ số, dấu chấm, gạch dưới hoặc gạch nối."})
	}
	if length := len([]rune(input.Name)); length < 2 || length > 255 {
		violations = append(violations, FieldViolation{Field: "name", Message: "Phải có từ 2 đến 255 ký tự."})
	}
	if len([]rune(input.TaxCode)) > 50 {
		violations = append(violations, FieldViolation{Field: "taxCode", Message: "Không được vượt quá 50 ký tự."})
	}
	if len([]rune(input.ContactName)) > 255 {
		violations = append(violations, FieldViolation{Field: "contactName", Message: "Không được vượt quá 255 ký tự."})
	}
	if input.Email != "" && (len(input.Email) > 255 || !emailPattern.MatchString(input.Email)) {
		violations = append(violations, FieldViolation{Field: "email", Message: "Phải là địa chỉ email hợp lệ."})
	}
	if len([]rune(input.Phone)) > 50 {
		violations = append(violations, FieldViolation{Field: "phone", Message: "Không được vượt quá 50 ký tự."})
	}
	if input.Status != "ACTIVE" && input.Status != "INACTIVE" {
		violations = append(violations, FieldViolation{Field: "status", Message: "Phải là ACTIVE (đang hoạt động) hoặc INACTIVE (ngừng hoạt động)."})
	}
	if input.RiskLevel != "LOW" && input.RiskLevel != "MEDIUM" && input.RiskLevel != "HIGH" {
		violations = append(violations, FieldViolation{Field: "riskLevel", Message: "Phải là LOW (thấp), MEDIUM (trung bình) hoặc HIGH (cao)."})
	}
	if len([]rune(input.Address)) > 1000 || len([]rune(input.BankName)) > 255 || len([]rune(input.BankAccountNumber)) > 100 || len([]rune(input.ContractReference)) > 100 || len([]rune(input.BusinessNote)) > 5000 {
		violations = append(violations, FieldViolation{Field: "supplierProfile", Message: "Có giá trị dài hơn giới hạn hệ thống hỗ trợ."})
	}
	if input.ContractExpiresOn != "" {
		if _, err := time.Parse(time.DateOnly, input.ContractExpiresOn); err != nil {
			violations = append(violations, FieldViolation{Field: "contractExpiresOn", Message: "Phải có định dạng YYYY-MM-DD."})
		}
	}
	if input.ComplianceStatus != "PENDING" && input.ComplianceStatus != "VERIFIED" && input.ComplianceStatus != "EXPIRED" && input.ComplianceStatus != "BLOCKED" {
		violations = append(violations, FieldViolation{Field: "complianceStatus", Message: "Phải là PENDING (chờ xác minh), VERIFIED (đã xác minh), EXPIRED (hết hiệu lực) hoặc BLOCKED (bị chặn)."})
	}
	if input.PerformanceScore != "" {
		score, ok := new(big.Rat).SetString(input.PerformanceScore)
		if !ok || score.Sign() < 0 || score.Cmp(big.NewRat(100, 1)) > 0 {
			violations = append(violations, FieldViolation{Field: "performanceScore", Message: "Phải nằm trong khoảng từ 0 đến 100."})
		}
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func ValidateUpdateSupplierInput(input *UpdateSupplierInput) error {
	err := ValidateSupplierInput(&input.SupplierInput)
	var violations []FieldViolation
	if validationError := (*ValidationError)(nil); errors.As(err, &validationError) {
		violations = append(violations, validationError.Violations...)
	} else if err != nil {
		return err
	}
	if input.ExpectedVersion < 1 {
		violations = append(violations, FieldViolation{Field: "expectedVersion", Message: "Phải lớn hơn hoặc bằng 1."})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func ValidateCreatePurchaseOrder(input *CreatePurchaseOrderInput) error {
	input.SupplierID = strings.TrimSpace(input.SupplierID)
	input.ExternalReference = strings.TrimSpace(input.ExternalReference)
	input.ExpectedDeliveryOn = strings.TrimSpace(input.ExpectedDeliveryOn)
	input.Note = strings.TrimSpace(input.Note)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	var violations []FieldViolation
	if !uuidPatternForDomain.MatchString(input.SupplierID) {
		violations = append(violations, FieldViolation{Field: "supplierId", Message: "Phải là UUID hợp lệ."})
	}
	if len([]rune(input.ExternalReference)) > 100 {
		violations = append(violations, FieldViolation{Field: "externalReference", Message: "Không được vượt quá 100 ký tự."})
	}
	deliveryDate, err := time.Parse(time.DateOnly, input.ExpectedDeliveryOn)
	if err != nil {
		violations = append(violations, FieldViolation{Field: "expectedDeliveryOn", Message: "Phải có định dạng YYYY-MM-DD."})
	} else if deliveryDate.Before(time.Now().UTC().Truncate(24 * time.Hour)) {
		violations = append(violations, FieldViolation{Field: "expectedDeliveryOn", Message: "Không được là ngày trong quá khứ."})
	}
	if len([]rune(input.Note)) > 2000 {
		violations = append(violations, FieldViolation{Field: "note", Message: "Không được vượt quá 2.000 ký tự."})
	}
	if !idempotencyPattern.MatchString(input.IdempotencyKey) {
		violations = append(violations, FieldViolation{Field: "Idempotency-Key", Message: "Phải có từ 8 đến 255 ký tự ASCII an toàn."})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func ValidateConfirmReceipt(input *ConfirmReceiptInput) error {
	input.ActualDeliveryOn = strings.TrimSpace(input.ActualDeliveryOn)
	var violations []FieldViolation
	if input.ExpectedVersion < 1 {
		violations = append(violations, FieldViolation{Field: "expectedVersion", Message: "Phải lớn hơn hoặc bằng 1."})
	}
	deliveryDate, err := time.Parse(time.DateOnly, input.ActualDeliveryOn)
	if err != nil {
		violations = append(violations, FieldViolation{Field: "actualDeliveryOn", Message: "Phải có định dạng YYYY-MM-DD."})
	} else if deliveryDate.After(time.Now().UTC().Truncate(24 * time.Hour)) {
		violations = append(violations, FieldViolation{Field: "actualDeliveryOn", Message: "Không được là ngày trong tương lai."})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func ValidateInvoiceInput(input *InvoiceInput) error {
	input.PurchaseOrderID = strings.TrimSpace(input.PurchaseOrderID)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	violations := validateInvoiceFields(
		&input.InvoiceNumber, &input.IssuedOn, &input.DueOn,
		&input.Amount, &input.Currency, &input.Note,
	)
	if !uuidPatternForDomain.MatchString(input.PurchaseOrderID) {
		violations = append(violations, FieldViolation{Field: "purchaseOrderId", Message: "Phải là UUID hợp lệ."})
	}
	if !idempotencyPattern.MatchString(input.IdempotencyKey) {
		violations = append(violations, FieldViolation{Field: "Idempotency-Key", Message: "Phải có từ 8 đến 255 ký tự ASCII an toàn."})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func ValidateUpdateInvoiceInput(input *UpdateInvoiceInput) error {
	violations := validateInvoiceFields(
		&input.InvoiceNumber, &input.IssuedOn, &input.DueOn,
		&input.Amount, &input.Currency, &input.Note,
	)
	if input.ExpectedVersion < 1 {
		violations = append(violations, FieldViolation{Field: "expectedVersion", Message: "Phải lớn hơn hoặc bằng 1."})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func ValidateInvoiceActionInput(input *InvoiceActionInput) error {
	input.Action = strings.ToUpper(strings.TrimSpace(input.Action))
	input.Comment = strings.TrimSpace(input.Comment)
	input.PaymentReference = strings.TrimSpace(input.PaymentReference)
	input.PaidOn = strings.TrimSpace(input.PaidOn)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	var violations []FieldViolation
	if input.Action != "VERIFY" && input.Action != "DISPUTE" &&
		input.Action != "REOPEN" && input.Action != "MARK_PAID" {
		violations = append(violations, FieldViolation{Field: "action", Message: "Phải là VERIFY (xác minh), DISPUTE (đối soát), REOPEN (mở lại) hoặc MARK_PAID (đánh dấu đã thanh toán)."})
	}
	if input.ExpectedVersion < 1 {
		violations = append(violations, FieldViolation{Field: "expectedVersion", Message: "Phải lớn hơn hoặc bằng 1."})
	}
	if len([]rune(input.Comment)) > 2000 {
		violations = append(violations, FieldViolation{Field: "comment", Message: "Không được vượt quá 2.000 ký tự."})
	}
	if input.Action == "DISPUTE" && input.Comment == "" {
		violations = append(violations, FieldViolation{Field: "comment", Message: "Bắt buộc khi đưa hóa đơn vào đối soát."})
	}
	if input.Action == "MARK_PAID" {
		if length := len([]rune(input.PaymentReference)); length < 2 || length > 100 || strings.ContainsAny(input.PaymentReference, "\r\n") {
			violations = append(violations, FieldViolation{Field: "paymentReference", Message: "Phải có từ 2 đến 100 ký tự trên một dòng."})
		}
		paidDate, err := time.Parse(time.DateOnly, input.PaidOn)
		if err != nil {
			violations = append(violations, FieldViolation{Field: "paidOn", Message: "Phải có định dạng YYYY-MM-DD."})
		} else if paidDate.After(time.Now().UTC().Truncate(24 * time.Hour)) {
			violations = append(violations, FieldViolation{Field: "paidOn", Message: "Không được là ngày trong tương lai."})
		}
	} else if input.PaymentReference != "" || input.PaidOn != "" {
		violations = append(violations, FieldViolation{Field: "paymentReference", Message: "Chỉ được dùng khi đánh dấu đã thanh toán (MARK_PAID)."})
	}
	if !idempotencyPattern.MatchString(input.IdempotencyKey) {
		violations = append(violations, FieldViolation{Field: "Idempotency-Key", Message: "Phải có từ 8 đến 255 ký tự ASCII an toàn."})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func validateInvoiceFields(
	invoiceNumber, issuedOn, dueOn, amount, currency, note *string,
) []FieldViolation {
	*invoiceNumber = strings.ToUpper(strings.TrimSpace(*invoiceNumber))
	*issuedOn = strings.TrimSpace(*issuedOn)
	*dueOn = strings.TrimSpace(*dueOn)
	*amount = strings.TrimSpace(*amount)
	*currency = strings.ToUpper(strings.TrimSpace(*currency))
	*note = strings.TrimSpace(*note)
	var violations []FieldViolation
	if !invoiceNumberPattern.MatchString(*invoiceNumber) {
		violations = append(violations, FieldViolation{Field: "invoiceNumber", Message: "Phải có từ 2 đến 100 chữ cái viết hoa, chữ số, dấu chấm, gạch chéo, gạch dưới hoặc gạch nối."})
	}
	issuedDate, issuedErr := time.Parse(time.DateOnly, *issuedOn)
	if issuedErr != nil {
		violations = append(violations, FieldViolation{Field: "issuedOn", Message: "Phải có định dạng YYYY-MM-DD."})
	} else if issuedDate.After(time.Now().UTC().Truncate(24 * time.Hour)) {
		violations = append(violations, FieldViolation{Field: "issuedOn", Message: "Không được là ngày trong tương lai."})
	}
	dueDate, dueErr := time.Parse(time.DateOnly, *dueOn)
	if dueErr != nil {
		violations = append(violations, FieldViolation{Field: "dueOn", Message: "Phải có định dạng YYYY-MM-DD."})
	} else if issuedErr == nil && dueDate.Before(issuedDate) {
		violations = append(violations, FieldViolation{Field: "dueOn", Message: "Không được trước ngày phát hành hóa đơn."})
	}
	if number, valid := decimal(*amount, unitPricePattern, true); !valid || number.Cmp(maxMoney) > 0 {
		violations = append(violations, FieldViolation{Field: "amount", Message: "Phải là số dương trong phạm vi tiền tệ hệ thống hỗ trợ."})
	}
	if !currencyPattern.MatchString(*currency) {
		violations = append(violations, FieldViolation{Field: "currency", Message: "Phải là mã tiền tệ gồm 3 chữ cái viết hoa."})
	}
	if len([]rune(*note)) > 2000 {
		violations = append(violations, FieldViolation{Field: "note", Message: "Không được vượt quá 2.000 ký tự."})
	}
	return violations
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
