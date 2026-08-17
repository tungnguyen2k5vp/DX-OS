package procurement

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/dx-os-lab/dx-os/backend/internal/notifications"
	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/jackc/pgx/v5"
)

type RecordPaymentInput struct {
	ExpectedVersion  int64
	Amount           string
	PaidOn           string
	PaymentReference string
	Note             string
	IdempotencyKey   string
	CorrelationID    string
}

type InvoicePayment struct {
	ID               string    `json:"id"`
	Amount           string    `json:"amount"`
	PaidOn           string    `json:"paidOn"`
	PaymentReference string    `json:"paymentReference"`
	Note             string    `json:"note,omitempty"`
	CreatedBy        string    `json:"createdBy"`
	CreatedAt        time.Time `json:"createdAt"`
}

type InvoicePaymentList struct {
	Items []InvoicePayment `json:"items"`
	Total int              `json:"total"`
}

func ValidateRecordPayment(input *RecordPaymentInput) error {
	input.Amount = strings.TrimSpace(input.Amount)
	input.PaidOn = strings.TrimSpace(input.PaidOn)
	input.PaymentReference = strings.TrimSpace(input.PaymentReference)
	input.Note = strings.TrimSpace(input.Note)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	var violations []FieldViolation
	if input.ExpectedVersion < 1 {
		violations = append(violations, FieldViolation{Field: "expectedVersion", Message: "must be greater than zero"})
	}
	if !unitPricePattern.MatchString(input.Amount) || mustRat(input.Amount).Sign() <= 0 {
		violations = append(violations, FieldViolation{Field: "amount", Message: "must be a positive monetary amount"})
	}
	paidOn, err := time.Parse(time.DateOnly, input.PaidOn)
	if err != nil {
		violations = append(violations, FieldViolation{Field: "paidOn", Message: "must use YYYY-MM-DD format"})
	} else if paidOn.After(time.Now().UTC().Truncate(24 * time.Hour)) {
		violations = append(violations, FieldViolation{Field: "paidOn", Message: "must not be in the future"})
	}
	if length := len([]rune(input.PaymentReference)); length < 2 || length > 100 || strings.ContainsAny(input.PaymentReference, "\r\n") {
		violations = append(violations, FieldViolation{Field: "paymentReference", Message: "must contain between 2 and 100 characters on one line"})
	}
	if len([]rune(input.Note)) > 2000 {
		violations = append(violations, FieldViolation{Field: "note", Message: "must not exceed 2000 characters"})
	}
	if !idempotencyPattern.MatchString(input.IdempotencyKey) {
		violations = append(violations, FieldViolation{Field: "Idempotency-Key", Message: "must contain 8 to 255 safe ASCII characters"})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func (s *Store) RecordInvoicePayment(ctx context.Context, principal auth.Principal, invoiceID string, input RecordPaymentInput) (InvoiceBoardItem, error) {
	if !hasRole(principal.Roles, "finance") || hasRole(principal.Roles, "auditor") {
		return InvoiceBoardItem{}, ErrForbidden
	}
	if err := ValidateRecordPayment(&input); err != nil {
		return InvoiceBoardItem{}, err
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return InvoiceBoardItem{}, fmt.Errorf("begin invoice payment: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return InvoiceBoardItem{}, err
	}
	var status, organizationID, amount, paidAmount, requestID, requesterID, departmentID, requestCode, requestTitle string
	var version int64
	err = tx.QueryRow(ctx, `
		SELECT pi.status, pi.version, pi.organization_id, pi.amount::text, pi.paid_amount::text,
			pr.id, pr.requester_id, pr.department_id, pr.request_code, pr.title
		FROM purchase_invoices pi
		JOIN purchase_orders po ON po.id = pi.purchase_order_id
		JOIN purchase_requests pr ON pr.id = po.purchase_request_id
		WHERE pi.id = $1 FOR UPDATE OF pi
	`, invoiceID).Scan(&status, &version, &organizationID, &amount, &paidAmount, &requestID, &requesterID, &departmentID, &requestCode, &requestTitle)
	if errors.Is(err, pgx.ErrNoRows) || organizationID != user.OrganizationID {
		return InvoiceBoardItem{}, ErrInvoiceNotFound
	}
	if err != nil {
		return InvoiceBoardItem{}, fmt.Errorf("lock invoice payment: %w", err)
	}
	var existingInvoiceID string
	err = tx.QueryRow(ctx, `SELECT invoice_id FROM invoice_payments WHERE idempotency_key=$1`, input.IdempotencyKey).Scan(&existingInvoiceID)
	if err == nil {
		if existingInvoiceID != invoiceID {
			return InvoiceBoardItem{}, ErrIdempotencyConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return InvoiceBoardItem{}, fmt.Errorf("commit payment replay: %w", err)
		}
		return s.loadInvoiceItem(ctx, invoiceID, true)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return InvoiceBoardItem{}, fmt.Errorf("check payment idempotency: %w", err)
	}
	if version != input.ExpectedVersion {
		return InvoiceBoardItem{}, ErrInvoiceVersion
	}
	if status != "VERIFIED" {
		return InvoiceBoardItem{}, ErrInvalidInvoiceAction
	}
	remaining := new(big.Rat).Sub(mustRat(amount), mustRat(paidAmount))
	if mustRat(input.Amount).Cmp(remaining) > 0 {
		return InvoiceBoardItem{}, &ValidationError{Violations: []FieldViolation{{Field: "amount", Message: "must not exceed the remaining invoice balance"}}}
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO invoice_payments (
			invoice_id, amount, paid_on, payment_reference, note, created_by,
			correlation_id, idempotency_key
		) VALUES ($1, $2::numeric, $3::date, $4, NULLIF($5,''), $6, NULLIF($7,''), $8)
	`, invoiceID, input.Amount, input.PaidOn, input.PaymentReference, input.Note, user.ID, input.CorrelationID, input.IdempotencyKey)
	if err != nil {
		if isUniqueViolation(err) {
			return InvoiceBoardItem{}, ErrIdempotencyConflict
		}
		return InvoiceBoardItem{}, fmt.Errorf("insert invoice payment: %w", err)
	}
	newPaid := new(big.Rat).Add(mustRat(paidAmount), mustRat(input.Amount))
	fullyPaid := newPaid.Cmp(mustRat(amount)) == 0
	toStatus := "VERIFIED"
	eventType := "INVOICE_PARTIAL_PAYMENT_RECORDED"
	if fullyPaid {
		toStatus = "PAID"
		eventType = "INVOICE_PAID"
	}
	_, err = tx.Exec(ctx, `
		UPDATE purchase_invoices
		SET paid_amount=$2::numeric, status=$3::varchar(20),
			paid_by=CASE WHEN $3::text='PAID' THEN $4::uuid ELSE paid_by END,
			paid_on=CASE WHEN $3::text='PAID' THEN $5::date ELSE paid_on END,
			payment_reference=CASE WHEN $3::text='PAID' THEN $6::text ELSE payment_reference END,
			version=version+1, updated_at=now()
		WHERE id=$1
	`, invoiceID, newPaid.FloatString(4), toStatus, user.ID, input.PaidOn, input.PaymentReference)
	if err != nil {
		return InvoiceBoardItem{}, fmt.Errorf("update invoice payment balance: %w", err)
	}
	if err = insertInvoiceEvent(ctx, tx, invoiceID, eventType, status, toStatus, user.ID, principal.Roles, input.Note, input.CorrelationID, input.IdempotencyKey); err != nil {
		return InvoiceBoardItem{}, err
	}
	if err = insertResourceAudit(ctx, tx, "purchase_invoice", invoiceID, eventType, user.ID, principal.Roles, status, toStatus, input.CorrelationID); err != nil {
		return InvoiceBoardItem{}, err
	}
	if err = notifications.Queue(ctx, tx, notifications.QueueInput{EventType: eventType, ResourceType: "purchase_request", ResourceID: requestID, OrganizationID: organizationID, DepartmentID: departmentID, RecipientUserID: requesterID, ActorID: user.ID, Title: "Cập nhật thanh toán " + requestCode, Body: requestTitle + " - " + input.Amount}); err != nil {
		return InvoiceBoardItem{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return InvoiceBoardItem{}, fmt.Errorf("commit invoice payment: %w", err)
	}
	return s.loadInvoiceItem(ctx, invoiceID, true)
}

func (s *Store) ListInvoicePayments(ctx context.Context, principal auth.Principal, invoiceID string) (InvoicePaymentList, error) {
	if !hasRole(principal.Roles, "finance") && !hasRole(principal.Roles, "auditor") {
		return InvoicePaymentList{}, ErrForbidden
	}
	user, err := s.ensureUser(ctx, principal)
	if err != nil {
		return InvoicePaymentList{}, err
	}
	var organizationID string
	if err = s.database.QueryRow(ctx, `SELECT organization_id FROM purchase_invoices WHERE id=$1`, invoiceID).Scan(&organizationID); errors.Is(err, pgx.ErrNoRows) || organizationID != user.OrganizationID {
		return InvoicePaymentList{}, ErrInvoiceNotFound
	}
	if err != nil {
		return InvoicePaymentList{}, fmt.Errorf("check invoice payment scope: %w", err)
	}
	rows, err := s.database.Query(ctx, `
		SELECT ip.id, ip.amount::text, ip.paid_on::text, ip.payment_reference,
			COALESCE(ip.note,''), u.display_name, ip.created_at
		FROM invoice_payments ip JOIN users u ON u.id=ip.created_by
		WHERE ip.invoice_id=$1 ORDER BY ip.created_at DESC, ip.id DESC
	`, invoiceID)
	if err != nil {
		return InvoicePaymentList{}, fmt.Errorf("list invoice payments: %w", err)
	}
	defer rows.Close()
	result := InvoicePaymentList{Items: make([]InvoicePayment, 0)}
	for rows.Next() {
		var item InvoicePayment
		if err = rows.Scan(&item.ID, &item.Amount, &item.PaidOn, &item.PaymentReference, &item.Note, &item.CreatedBy, &item.CreatedAt); err != nil {
			return InvoicePaymentList{}, fmt.Errorf("scan invoice payment: %w", err)
		}
		result.Items = append(result.Items, item)
	}
	if err = rows.Err(); err != nil {
		return InvoicePaymentList{}, fmt.Errorf("iterate invoice payments: %w", err)
	}
	result.Total = len(result.Items)
	return result, nil
}
