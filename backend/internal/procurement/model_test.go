package procurement

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
)

func TestValidateCreateNormalizesAndAcceptsValidInput(t *testing.T) {
	input := CreateInput{
		Title:      "  Laptop for design team  ",
		Reason:     "  Required for the approved design workload.  ",
		Currency:   "vnd",
		CostCenter: "  CC-DESIGN  ",
		Items: []CreateItemInput{
			{
				Description: "  Developer laptop  ",
				Quantity:    "2",
				Unit:        "  unit  ",
				UnitPrice:   "25000000.0000",
			},
		},
	}

	if err := ValidateCreate(&input); err != nil {
		t.Fatalf("expected valid input, got %v", err)
	}
	if input.Title != "Laptop for design team" || input.Currency != "VND" {
		t.Fatalf("input was not normalized: %#v", input)
	}
	if input.Items[0].Description != "Developer laptop" {
		t.Fatalf("item was not normalized: %#v", input.Items[0])
	}
}

func TestValidateCreateReportsFieldViolations(t *testing.T) {
	input := CreateInput{
		Title:      "x",
		Reason:     "short",
		Currency:   "12",
		CostCenter: "",
		Items: []CreateItemInput{
			{Description: "", Quantity: "0", Unit: "", UnitPrice: "-1"},
		},
	}

	err := ValidateCreate(&input)
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if len(validationError.Violations) != 8 {
		t.Fatalf("expected 8 violations, got %d: %#v", len(validationError.Violations), validationError.Violations)
	}
}

func TestScopeForUsesLeastBroadBusinessRole(t *testing.T) {
	tests := []struct {
		name      string
		roles     []string
		wantScope ScopeKind
		wantErr   error
	}{
		{name: "employee", roles: []string{"employee"}, wantScope: ScopeOwn},
		{name: "manager", roles: []string{"employee", "department_manager"}, wantScope: ScopeDepartment},
		{name: "finance", roles: []string{"finance"}, wantScope: ScopeFinance},
		{name: "auditor", roles: []string{"auditor"}, wantScope: ScopeAll},
		{name: "admin is not implicit superuser", roles: []string{"dx_admin"}, wantErr: ErrForbidden},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scope, err := ScopeFor(auth.Principal{Roles: test.roles})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error %v, got %v", test.wantErr, err)
			}
			if err == nil && scope != test.wantScope {
				t.Fatalf("expected scope %v, got %v", test.wantScope, scope)
			}
		})
	}
}

func TestCanCreateRequiresEmployeeOrManager(t *testing.T) {
	if !CanCreate(auth.Principal{Roles: []string{"employee"}}) {
		t.Fatal("employee should be allowed to create a purchase request")
	}
	if CanCreate(auth.Principal{Roles: []string{"auditor", "dx_admin"}}) {
		t.Fatal("auditor/admin should not implicitly create a purchase request")
	}
}

func TestValidateCreateRejectsMoneyOverflow(t *testing.T) {
	input := CreateInput{
		Title:      "Very expensive equipment",
		Reason:     "Valid business reason for exercising the numeric limit.",
		Currency:   "VND",
		CostCenter: "CC-GENERAL",
		Items: []CreateItemInput{
			{
				Description: "Equipment",
				Quantity:    "2",
				Unit:        "unit",
				UnitPrice:   "999999999999999.9999",
			},
		},
	}

	err := ValidateCreate(&input)
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if len(validationError.Violations) != 1 ||
		validationError.Violations[0].Field != "items[0].unitPrice" {
		t.Fatalf("unexpected violations: %#v", validationError.Violations)
	}
}

func TestDecideTransition(t *testing.T) {
	const (
		requesterID  = "requester"
		managerID    = "manager"
		financeID    = "finance"
		departmentID = "department"
		organization = "organization"
	)
	tests := []struct {
		name      string
		actor     ActorContext
		request   RequestContext
		action    Action
		want      Status
		wantError error
	}{
		{
			name:    "requester submits draft",
			actor:   ActorContext{UserID: requesterID},
			request: RequestContext{RequesterID: requesterID, Status: StatusDraft},
			action:  ActionSubmit,
			want:    StatusSubmitted,
		},
		{
			name:    "requester resubmits changes",
			actor:   ActorContext{UserID: requesterID},
			request: RequestContext{RequesterID: requesterID, Status: StatusChangesRequested},
			action:  ActionResubmit,
			want:    StatusSubmitted,
		},
		{
			name: "manager approves same department",
			actor: ActorContext{
				UserID: managerID, DepartmentID: departmentID,
				Roles: []string{"department_manager"},
			},
			request: RequestContext{
				RequesterID: requesterID, DepartmentID: departmentID, Status: StatusSubmitted,
			},
			action: ActionApprove,
			want:   StatusManagerApproved,
		},
		{
			name: "finance approves same organization",
			actor: ActorContext{
				UserID: financeID, OrganizationID: organization, Roles: []string{"finance"},
			},
			request: RequestContext{
				RequesterID: requesterID, OrganizationID: organization, Status: StatusManagerApproved,
			},
			action: ActionApprove,
			want:   StatusApproved,
		},
		{
			name: "requester cannot self approve as manager",
			actor: ActorContext{
				UserID: requesterID, DepartmentID: departmentID,
				Roles: []string{"department_manager"},
			},
			request: RequestContext{
				RequesterID: requesterID, DepartmentID: departmentID, Status: StatusSubmitted,
			},
			action:    ActionApprove,
			wantError: ErrForbidden,
		},
		{
			name: "manager outside department sees not found",
			actor: ActorContext{
				UserID: managerID, DepartmentID: "other",
				Roles: []string{"department_manager"},
			},
			request: RequestContext{
				RequesterID: requesterID, DepartmentID: departmentID, Status: StatusSubmitted,
			},
			action:    ActionApprove,
			wantError: ErrNotFound,
		},
		{
			name:      "cannot approve draft",
			actor:     ActorContext{UserID: requesterID},
			request:   RequestContext{RequesterID: requesterID, Status: StatusDraft},
			action:    ActionApprove,
			wantError: ErrInvalidTransition,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decision, err := DecideTransition(test.actor, test.request, test.action)
			if !errors.Is(err, test.wantError) {
				t.Fatalf("expected error %v, got %v", test.wantError, err)
			}
			if err == nil && decision.ToStatus != test.want {
				t.Fatalf("expected status %s, got %s", test.want, decision.ToStatus)
			}
		})
	}
}

func TestValidateTransitionRequiresCommentAndIdempotency(t *testing.T) {
	input := TransitionInput{
		Action:          ActionReject,
		ExpectedVersion: 1,
	}
	err := ValidateTransition(&input)
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if len(validationError.Violations) != 2 {
		t.Fatalf("expected comment and idempotency violations, got %#v", validationError.Violations)
	}
}

func TestValidateBudgetSummaryInputNormalizesValues(t *testing.T) {
	input := BudgetSummaryInput{CostCenter: "  CC-GENERAL  ", Currency: "vnd"}
	if err := ValidateBudgetSummaryInput(&input); err != nil {
		t.Fatalf("expected valid budget query, got %v", err)
	}
	if input.CostCenter != "CC-GENERAL" || input.Currency != "VND" {
		t.Fatalf("budget query was not normalized: %#v", input)
	}
}

func TestValidateBudgetSummaryInputRejectsMissingValues(t *testing.T) {
	input := BudgetSummaryInput{}
	err := ValidateBudgetSummaryInput(&input)
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if len(validationError.Violations) != 2 {
		t.Fatalf("expected cost center and currency violations, got %#v", validationError.Violations)
	}
}

func TestValidateAdjustBudgetInput(t *testing.T) {
	input := AdjustBudgetInput{
		AllocatedAmount: " 120000000000 ",
		ExpectedVersion: 2,
		Reason:          " Approved annual allocation increase. ",
		IdempotencyKey:  "budget-adjustment-0001",
	}
	if err := ValidateAdjustBudgetInput(&input); err != nil {
		t.Fatalf("expected valid adjustment, got %v", err)
	}
	if input.AllocatedAmount != "120000000000" ||
		input.Reason != "Approved annual allocation increase." {
		t.Fatalf("adjustment was not normalized: %#v", input)
	}
}

func TestValidateAdjustBudgetInputRejectsUnsafeValues(t *testing.T) {
	input := AdjustBudgetInput{
		AllocatedAmount: "-1",
		Reason:          "short",
	}
	err := ValidateAdjustBudgetInput(&input)
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if len(validationError.Violations) != 4 {
		t.Fatalf("expected four violations, got %#v", validationError.Violations)
	}
}

func TestValidateAttachmentAcceptsQuotationPDF(t *testing.T) {
	input := UploadAttachmentInput{
		DocumentType: DocumentTypeQuotation,
		FileName:     "  bao-gia.pdf  ",
		ContentType:  "application/pdf",
		Content:      []byte("%PDF-test"),
	}
	if err := ValidateAttachment(&input); err != nil {
		t.Fatalf("expected valid attachment, got %v", err)
	}
	if input.FileName != "bao-gia.pdf" {
		t.Fatalf("filename was not normalized: %q", input.FileName)
	}
}

func TestValidateAttachmentRejectsUnsafeFile(t *testing.T) {
	input := UploadAttachmentInput{
		DocumentType: "EXECUTABLE",
		FileName:     "../payload.exe",
		ContentType:  "application/octet-stream",
		Content:      nil,
	}
	err := ValidateAttachment(&input)
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if len(validationError.Violations) != 4 {
		t.Fatalf("expected four violations, got %#v", validationError.Violations)
	}
}

func TestValidateCommentNormalizesBody(t *testing.T) {
	input := CommentInput{Body: "  Please check the quotation.  "}
	if err := ValidateComment(&input); err != nil {
		t.Fatalf("expected valid comment, got %v", err)
	}
	if input.Body != "Please check the quotation." {
		t.Fatalf("comment was not normalized: %q", input.Body)
	}
}

func TestValidateCommentRejectsEmptyAndOversizedBody(t *testing.T) {
	for _, body := range []string{"   ", strings.Repeat("a", 2001)} {
		input := CommentInput{Body: body}
		var validationError *ValidationError
		if err := ValidateComment(&input); !errors.As(err, &validationError) {
			t.Fatalf("expected ValidationError for length %d, got %v", len(body), err)
		}
	}
}

func TestValidateSupplierInputNormalizesValues(t *testing.T) {
	input := SupplierInput{
		Code: "  vendor-01 ", Name: "  Demo Vendor  ", TaxCode: " TAX-01 ",
		Email: " SALES@EXAMPLE.COM ", Status: "active", RiskLevel: "medium",
	}
	if err := ValidateSupplierInput(&input); err != nil {
		t.Fatalf("expected valid supplier, got %v", err)
	}
	if input.Code != "VENDOR-01" || input.Name != "Demo Vendor" ||
		input.Email != "sales@example.com" || input.RiskLevel != "MEDIUM" {
		t.Fatalf("supplier was not normalized: %#v", input)
	}
}

func TestValidateSupplierInputRejectsUnsafeValues(t *testing.T) {
	input := SupplierInput{Code: "!", Name: "x", Email: "invalid", Status: "UNKNOWN", RiskLevel: "NONE"}
	var validationError *ValidationError
	if err := ValidateSupplierInput(&input); !errors.As(err, &validationError) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if len(validationError.Violations) != 5 {
		t.Fatalf("expected five supplier violations, got %#v", validationError.Violations)
	}
}

func TestValidateCreatePurchaseOrder(t *testing.T) {
	input := CreatePurchaseOrderInput{
		SupplierID:         "00000000-0000-4000-8000-000000000801",
		ExpectedDeliveryOn: time.Now().UTC().AddDate(0, 0, 2).Format(time.DateOnly),
		IdempotencyKey:     "purchase-order-0001",
	}
	if err := ValidateCreatePurchaseOrder(&input); err != nil {
		t.Fatalf("expected valid purchase order, got %v", err)
	}
}

func TestValidateInvoiceInputs(t *testing.T) {
	create := InvoiceInput{
		PurchaseOrderID: "00000000-0000-4000-8000-000000000802",
		InvoiceNumber:   " inv-2026/001 ", IssuedOn: time.Now().UTC().Format(time.DateOnly),
		DueOn:  time.Now().UTC().AddDate(0, 1, 0).Format(time.DateOnly),
		Amount: "27000000", Currency: "vnd", IdempotencyKey: "invoice-create-0001",
	}
	if err := ValidateInvoiceInput(&create); err != nil {
		t.Fatalf("valid invoice rejected: %v", err)
	}
	if create.InvoiceNumber != "INV-2026/001" || create.Currency != "VND" {
		t.Fatalf("invoice fields were not normalized: %+v", create)
	}

	action := InvoiceActionInput{
		Action: "mark_paid", ExpectedVersion: 2, PaymentReference: "BANK-001",
		PaidOn: time.Now().UTC().Format(time.DateOnly), IdempotencyKey: "invoice-payment-0001",
	}
	if err := ValidateInvoiceActionInput(&action); err != nil {
		t.Fatalf("valid payment action rejected: %v", err)
	}

	invalid := InvoiceActionInput{Action: "DISPUTE", ExpectedVersion: 1, IdempotencyKey: "invoice-dispute-0001"}
	if err := ValidateInvoiceActionInput(&invalid); err == nil {
		t.Fatal("dispute without a comment must be rejected")
	}
}

func TestInvoiceMatchStatus(t *testing.T) {
	cases := []struct {
		name, orderStatus, invoiceAmount, invoiceCurrency, orderAmount, orderCurrency, expected string
	}{
		{"waiting receipt", "ORDERED", "100.0000", "VND", "100.0000", "VND", "WAITING_RECEIPT"},
		{"currency mismatch", "RECEIVED", "100.0000", "USD", "100.0000", "VND", "CURRENCY_MISMATCH"},
		{"partial invoice", "RECEIVED", "99.0000", "VND", "100.0000", "VND", "PARTIAL_MATCH"},
		{"amount exceeds order", "RECEIVED", "101.0000", "VND", "100.0000", "VND", "AMOUNT_MISMATCH"},
		{"matched decimal representations", "RECEIVED", "100", "VND", "100.0000", "VND", "MATCHED"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			actual := invoiceMatchStatus(test.orderStatus, test.invoiceAmount, test.invoiceCurrency, test.orderAmount, test.orderCurrency)
			if actual != test.expected {
				t.Fatalf("expected %s, got %s", test.expected, actual)
			}
		})
	}
}

func TestValidateEnterpriseCompletionInputs(t *testing.T) {
	receipt := RecordReceiptInput{
		ExpectedVersion: 2,
		Outcome:         "partial",
		ReceivedOn:      time.Now().UTC().Format(time.DateOnly),
		Note:            "First shipment received.",
		IdempotencyKey:  "receipt-partial-0001",
		Items: []ReceiptItemInput{{
			PurchaseRequestItemID: "00000000-0000-4000-8000-000000000811",
			QuantityReceived:      "2.0000",
			Condition:             "accepted",
		}},
	}
	if err := ValidateRecordReceipt(&receipt); err != nil {
		t.Fatalf("valid receipt rejected: %v", err)
	}
	if receipt.Outcome != "PARTIAL" || receipt.Items[0].Condition != "ACCEPTED" {
		t.Fatalf("receipt was not normalized: %#v", receipt)
	}

	payment := RecordPaymentInput{
		ExpectedVersion:  3,
		Amount:           "5000000",
		PaidOn:           time.Now().UTC().Format(time.DateOnly),
		PaymentReference: " bank-0001 ",
		Note:             "First installment.",
		IdempotencyKey:   "payment-partial-0001",
	}
	if err := ValidateRecordPayment(&payment); err != nil {
		t.Fatalf("valid payment rejected: %v", err)
	}
	if payment.PaymentReference != "bank-0001" {
		t.Fatalf("payment reference was not normalized: %#v", payment)
	}

	auditCase := AuditCaseInput{
		Title:       "Delivery control",
		Description: "Review the damaged delivery exception.",
		Severity:    "high",
		Status:      "open",
	}
	if err := validateAuditCase(&auditCase, false); err != nil {
		t.Fatalf("valid audit case rejected: %v", err)
	}
	if auditCase.Severity != "HIGH" || auditCase.Status != "OPEN" {
		t.Fatalf("audit case was not normalized: %#v", auditCase)
	}
}
