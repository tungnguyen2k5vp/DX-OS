package procurement

import (
	"context"
	"errors"
	"fmt"

	"github.com/dx-os-lab/dx-os/backend/internal/notifications"
	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/jackc/pgx/v5"
)

func (s *Store) InvoiceBoard(ctx context.Context, principal auth.Principal) (InvoiceBoard, error) {
	if !hasRole(principal.Roles, "finance") && !hasRole(principal.Roles, "auditor") {
		return InvoiceBoard{}, ErrForbidden
	}
	user, err := s.ensureUser(ctx, principal)
	if err != nil {
		return InvoiceBoard{}, err
	}
	isAuditor := hasRole(principal.Roles, "auditor")
	canManage := hasRole(principal.Roles, "finance") && !isAuditor
	rows, err := s.database.Query(ctx, `
		SELECT
			po.id, pr.id, pr.request_code, pr.title, ru.display_name, d.name,
			s.id, s.code, s.name, po.order_code, po.status, pr.total_amount::text,
			pr.currency, po.actual_delivery_on::text,
			pi.id, pi.invoice_number, pi.issued_on::text, pi.due_on::text,
			pi.amount::text, pi.currency, pi.status,
			CASE
				WHEN pi.id IS NULL THEN 'NOT_RECORDED'
				WHEN po.status <> 'RECEIVED' THEN 'WAITING_RECEIPT'
				WHEN pi.currency <> pr.currency THEN 'CURRENCY_MISMATCH'
				WHEN pi.amount <> pr.total_amount THEN 'AMOUNT_MISMATCH'
				ELSE 'MATCHED'
			END,
			pi.note, COALESCE(pi.version, 0), pi.payment_reference, pi.paid_on::text,
			pi.created_at, pi.updated_at,
			COALESCE(pi.status <> 'PAID' AND pi.due_on < CURRENT_DATE, false),
			$3::boolean
		FROM purchase_orders po
		JOIN purchase_requests pr ON pr.id = po.purchase_request_id
		JOIN users ru ON ru.id = pr.requester_id
		JOIN departments d ON d.id = pr.department_id
		JOIN suppliers s ON s.id = po.supplier_id
		LEFT JOIN purchase_invoices pi ON pi.purchase_order_id = po.id
		WHERE $1::boolean OR po.organization_id = $2
		ORDER BY
			CASE WHEN pi.status <> 'PAID' AND pi.due_on < CURRENT_DATE THEN 1 ELSE 2 END,
			CASE WHEN pi.id IS NULL THEN 1 WHEN pi.status = 'DISPUTED' THEN 2
				WHEN pi.status = 'RECORDED' THEN 3 WHEN pi.status = 'VERIFIED' THEN 4 ELSE 5 END,
			COALESCE(pi.due_on, CURRENT_DATE + 3650), po.updated_at DESC
		LIMIT 300
	`, isAuditor, user.OrganizationID, canManage)
	if err != nil {
		return InvoiceBoard{}, fmt.Errorf("list invoice board: %w", err)
	}
	defer rows.Close()
	result := InvoiceBoard{Items: make([]InvoiceBoardItem, 0), CanManage: canManage}
	for rows.Next() {
		var item InvoiceBoardItem
		if err = rows.Scan(invoiceBoardDestinations(&item)...); err != nil {
			return InvoiceBoard{}, fmt.Errorf("scan invoice board item: %w", err)
		}
		result.Items = append(result.Items, item)
		result.Total++
		switch {
		case item.InvoiceID == nil:
			result.AwaitingInvoiceCount++
		case item.InvoiceStatus != nil && *item.InvoiceStatus == "PAID":
			result.PaidCount++
		case item.InvoiceStatus != nil && *item.InvoiceStatus == "VERIFIED":
			result.ReadyToPayCount++
		default:
			result.NeedsReviewCount++
		}
		if item.PaymentOverdue {
			result.OverdueCount++
		}
	}
	if err = rows.Err(); err != nil {
		return InvoiceBoard{}, fmt.Errorf("iterate invoice board: %w", err)
	}
	return result, nil
}

func (s *Store) CreateInvoice(
	ctx context.Context,
	principal auth.Principal,
	input InvoiceInput,
) (InvoiceBoardItem, error) {
	if !hasRole(principal.Roles, "finance") || hasRole(principal.Roles, "auditor") {
		return InvoiceBoardItem{}, ErrForbidden
	}
	if err := ValidateInvoiceInput(&input); err != nil {
		return InvoiceBoardItem{}, err
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return InvoiceBoardItem{}, fmt.Errorf("begin create invoice: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return InvoiceBoardItem{}, err
	}
	var existingID, existingOrderID string
	err = tx.QueryRow(ctx, `
		SELECT id, purchase_order_id FROM purchase_invoices WHERE idempotency_key = $1
	`, input.IdempotencyKey).Scan(&existingID, &existingOrderID)
	switch {
	case err == nil:
		if existingOrderID != input.PurchaseOrderID {
			return InvoiceBoardItem{}, ErrInvoiceConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return InvoiceBoardItem{}, fmt.Errorf("commit invoice replay: %w", err)
		}
		return s.loadInvoiceItem(ctx, existingID, true)
	case !errors.Is(err, pgx.ErrNoRows):
		return InvoiceBoardItem{}, fmt.Errorf("check invoice idempotency: %w", err)
	}
	var organizationID, requesterID, departmentID, requestID, requestCode, requestTitle string
	err = tx.QueryRow(ctx, `
		SELECT po.organization_id, pr.requester_id, pr.department_id, pr.id, pr.request_code, pr.title
		FROM purchase_orders po
		JOIN purchase_requests pr ON pr.id = po.purchase_request_id
		WHERE po.id = $1
		FOR UPDATE OF po
	`, input.PurchaseOrderID).Scan(
		&organizationID, &requesterID, &departmentID, &requestID, &requestCode, &requestTitle,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return InvoiceBoardItem{}, ErrPurchaseOrderNotFound
	}
	if err != nil {
		return InvoiceBoardItem{}, fmt.Errorf("lock invoice purchase order: %w", err)
	}
	if organizationID != user.OrganizationID {
		return InvoiceBoardItem{}, ErrPurchaseOrderNotFound
	}
	var invoiceID string
	err = tx.QueryRow(ctx, `
		INSERT INTO purchase_invoices (
			organization_id, purchase_order_id, invoice_number, issued_on, due_on,
			amount, currency, note, created_by, idempotency_key
		)
		VALUES ($1, $2, $3, $4::date, $5::date, $6::numeric, $7, NULLIF($8, ''), $9, $10)
		RETURNING id
	`, organizationID, input.PurchaseOrderID, input.InvoiceNumber, input.IssuedOn,
		input.DueOn, input.Amount, input.Currency, input.Note, user.ID, input.IdempotencyKey).Scan(&invoiceID)
	if err != nil {
		if isUniqueViolation(err) {
			return InvoiceBoardItem{}, ErrInvoiceConflict
		}
		return InvoiceBoardItem{}, fmt.Errorf("insert invoice: %w", err)
	}
	if err = insertInvoiceEvent(ctx, tx, invoiceID, "INVOICE_RECORDED", "", "RECORDED",
		user.ID, principal.Roles, input.Note, input.CorrelationID, ""); err != nil {
		return InvoiceBoardItem{}, err
	}
	if err = insertResourceAudit(ctx, tx, "purchase_invoice", invoiceID, "INVOICE_RECORDED",
		user.ID, principal.Roles, "", "RECORDED", input.CorrelationID); err != nil {
		return InvoiceBoardItem{}, err
	}
	if err = notifications.Queue(ctx, tx, notifications.QueueInput{
		EventType: "INVOICE_RECORDED", ResourceType: "purchase_request",
		ResourceID: requestID, OrganizationID: organizationID,
		DepartmentID: departmentID, RecipientUserID: requesterID, ActorID: user.ID,
		Title: "Hóa đơn mua sắm đã được ghi nhận", Body: requestCode + " - " + requestTitle,
	}); err != nil {
		return InvoiceBoardItem{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return InvoiceBoardItem{}, fmt.Errorf("commit create invoice: %w", err)
	}
	return s.loadInvoiceItem(ctx, invoiceID, true)
}

func (s *Store) UpdateInvoice(
	ctx context.Context,
	principal auth.Principal,
	invoiceID string,
	input UpdateInvoiceInput,
) (InvoiceBoardItem, error) {
	if !hasRole(principal.Roles, "finance") || hasRole(principal.Roles, "auditor") {
		return InvoiceBoardItem{}, ErrForbidden
	}
	if err := ValidateUpdateInvoiceInput(&input); err != nil {
		return InvoiceBoardItem{}, err
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return InvoiceBoardItem{}, fmt.Errorf("begin update invoice: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return InvoiceBoardItem{}, err
	}
	var organizationID, status string
	var version int64
	err = tx.QueryRow(ctx, `
		SELECT organization_id, status, version FROM purchase_invoices WHERE id = $1 FOR UPDATE
	`, invoiceID).Scan(&organizationID, &status, &version)
	if errors.Is(err, pgx.ErrNoRows) || organizationID != user.OrganizationID {
		return InvoiceBoardItem{}, ErrInvoiceNotFound
	}
	if err != nil {
		return InvoiceBoardItem{}, fmt.Errorf("lock invoice for update: %w", err)
	}
	if status != "RECORDED" && status != "DISPUTED" {
		return InvoiceBoardItem{}, ErrInvalidInvoiceAction
	}
	if version != input.ExpectedVersion {
		return InvoiceBoardItem{}, ErrInvoiceVersion
	}
	_, err = tx.Exec(ctx, `
		UPDATE purchase_invoices
		SET invoice_number = $2, issued_on = $3::date, due_on = $4::date,
			amount = $5::numeric, currency = $6, note = NULLIF($7, ''),
			status = 'RECORDED', verified_by = NULL, disputed_by = NULL,
			version = version + 1, updated_at = now()
		WHERE id = $1
	`, invoiceID, input.InvoiceNumber, input.IssuedOn, input.DueOn,
		input.Amount, input.Currency, input.Note)
	if err != nil {
		if isUniqueViolation(err) {
			return InvoiceBoardItem{}, ErrInvoiceConflict
		}
		return InvoiceBoardItem{}, fmt.Errorf("update invoice: %w", err)
	}
	if err = insertInvoiceEvent(ctx, tx, invoiceID, "INVOICE_UPDATED", status, "RECORDED",
		user.ID, principal.Roles, input.Note, input.CorrelationID, ""); err != nil {
		return InvoiceBoardItem{}, err
	}
	if err = insertResourceAudit(ctx, tx, "purchase_invoice", invoiceID, "INVOICE_UPDATED",
		user.ID, principal.Roles, status, "RECORDED", input.CorrelationID); err != nil {
		return InvoiceBoardItem{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return InvoiceBoardItem{}, fmt.Errorf("commit update invoice: %w", err)
	}
	return s.loadInvoiceItem(ctx, invoiceID, true)
}

func (s *Store) TransitionInvoice(
	ctx context.Context,
	principal auth.Principal,
	invoiceID string,
	input InvoiceActionInput,
) (InvoiceBoardItem, error) {
	if !hasRole(principal.Roles, "finance") || hasRole(principal.Roles, "auditor") {
		return InvoiceBoardItem{}, ErrForbidden
	}
	if err := ValidateInvoiceActionInput(&input); err != nil {
		return InvoiceBoardItem{}, err
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return InvoiceBoardItem{}, fmt.Errorf("begin invoice transition: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return InvoiceBoardItem{}, err
	}
	var replayInvoiceID, replayAction string
	err = tx.QueryRow(ctx, `
		SELECT invoice_id, event_type FROM invoice_events WHERE idempotency_key = $1
	`, input.IdempotencyKey).Scan(&replayInvoiceID, &replayAction)
	switch {
	case err == nil:
		if replayInvoiceID != invoiceID || replayAction != invoiceEventType(input.Action) {
			return InvoiceBoardItem{}, ErrIdempotencyConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return InvoiceBoardItem{}, fmt.Errorf("commit invoice transition replay: %w", err)
		}
		return s.loadInvoiceItem(ctx, invoiceID, true)
	case !errors.Is(err, pgx.ErrNoRows):
		return InvoiceBoardItem{}, fmt.Errorf("check invoice transition idempotency: %w", err)
	}
	var status, organizationID, orderStatus, invoiceAmount, invoiceCurrency, orderAmount, orderCurrency string
	var requesterID, departmentID, requestID, requestCode, requestTitle string
	var version int64
	err = tx.QueryRow(ctx, `
		SELECT pi.status, pi.version, pi.organization_id, po.status,
			pi.amount::text, pi.currency, pr.total_amount::text, pr.currency,
			pr.requester_id, pr.department_id, pr.id, pr.request_code, pr.title
		FROM purchase_invoices pi
		JOIN purchase_orders po ON po.id = pi.purchase_order_id
		JOIN purchase_requests pr ON pr.id = po.purchase_request_id
		WHERE pi.id = $1
		FOR UPDATE OF pi
	`, invoiceID).Scan(
		&status, &version, &organizationID, &orderStatus,
		&invoiceAmount, &invoiceCurrency, &orderAmount, &orderCurrency,
		&requesterID, &departmentID, &requestID, &requestCode, &requestTitle,
	)
	if errors.Is(err, pgx.ErrNoRows) || organizationID != user.OrganizationID {
		return InvoiceBoardItem{}, ErrInvoiceNotFound
	}
	if err != nil {
		return InvoiceBoardItem{}, fmt.Errorf("lock invoice transition: %w", err)
	}
	if version != input.ExpectedVersion {
		return InvoiceBoardItem{}, ErrInvoiceVersion
	}
	matchStatus := invoiceMatchStatus(orderStatus, invoiceAmount, invoiceCurrency, orderAmount, orderCurrency)
	toStatus := ""
	switch input.Action {
	case "VERIFY":
		if status != "RECORDED" {
			return InvoiceBoardItem{}, ErrInvalidInvoiceAction
		}
		if matchStatus != "MATCHED" {
			return InvoiceBoardItem{}, ErrInvoiceMismatch
		}
		toStatus = "VERIFIED"
	case "DISPUTE":
		if status != "RECORDED" {
			return InvoiceBoardItem{}, ErrInvalidInvoiceAction
		}
		toStatus = "DISPUTED"
	case "REOPEN":
		if status != "DISPUTED" {
			return InvoiceBoardItem{}, ErrInvalidInvoiceAction
		}
		toStatus = "RECORDED"
	case "MARK_PAID":
		if status != "VERIFIED" {
			return InvoiceBoardItem{}, ErrInvalidInvoiceAction
		}
		toStatus = "PAID"
	}
	_, err = tx.Exec(ctx, `
		UPDATE purchase_invoices
		SET status = $2::varchar(20),
			verified_by = CASE WHEN $2::text = 'VERIFIED' THEN $3::uuid ELSE verified_by END,
			disputed_by = CASE WHEN $2::text = 'DISPUTED' THEN $3::uuid WHEN $2::text = 'RECORDED' THEN NULL ELSE disputed_by END,
			paid_by = CASE WHEN $2::text = 'PAID' THEN $3::uuid ELSE paid_by END,
			payment_reference = CASE WHEN $2::text = 'PAID' THEN $4::text ELSE payment_reference END,
			paid_on = CASE WHEN $2::text = 'PAID' THEN $5::date ELSE paid_on END,
			version = version + 1, updated_at = now()
		WHERE id = $1
	`, invoiceID, toStatus, user.ID, input.PaymentReference, nullableDate(input.PaidOn))
	if err != nil {
		return InvoiceBoardItem{}, fmt.Errorf("update invoice transition: %w", err)
	}
	eventType := invoiceEventType(input.Action)
	if err = insertInvoiceEvent(ctx, tx, invoiceID, eventType, status, toStatus,
		user.ID, principal.Roles, input.Comment, input.CorrelationID, input.IdempotencyKey); err != nil {
		return InvoiceBoardItem{}, err
	}
	if err = insertResourceAudit(ctx, tx, "purchase_invoice", invoiceID, eventType,
		user.ID, principal.Roles, status, toStatus, input.CorrelationID); err != nil {
		return InvoiceBoardItem{}, err
	}
	if toStatus == "PAID" {
		if err = notifications.Queue(ctx, tx, notifications.QueueInput{
			EventType: eventType, ResourceType: "purchase_request", ResourceID: requestID,
			OrganizationID: organizationID, DepartmentID: departmentID,
			RecipientUserID: requesterID, ActorID: user.ID,
			Title: "Hóa đơn mua sắm đã thanh toán", Body: requestCode + " - " + requestTitle,
		}); err != nil {
			return InvoiceBoardItem{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return InvoiceBoardItem{}, fmt.Errorf("commit invoice transition: %w", err)
	}
	return s.loadInvoiceItem(ctx, invoiceID, true)
}

func (s *Store) loadInvoiceItem(ctx context.Context, invoiceID string, canManage bool) (InvoiceBoardItem, error) {
	var item InvoiceBoardItem
	err := s.database.QueryRow(ctx, `
		SELECT
			po.id, pr.id, pr.request_code, pr.title, ru.display_name, d.name,
			s.id, s.code, s.name, po.order_code, po.status, pr.total_amount::text,
			pr.currency, po.actual_delivery_on::text,
			pi.id, pi.invoice_number, pi.issued_on::text, pi.due_on::text,
			pi.amount::text, pi.currency, pi.status,
			CASE WHEN po.status <> 'RECEIVED' THEN 'WAITING_RECEIPT'
				WHEN pi.currency <> pr.currency THEN 'CURRENCY_MISMATCH'
				WHEN pi.amount <> pr.total_amount THEN 'AMOUNT_MISMATCH'
				ELSE 'MATCHED' END,
			pi.note, pi.version, pi.payment_reference, pi.paid_on::text,
			pi.created_at, pi.updated_at,
			(pi.status <> 'PAID' AND pi.due_on < CURRENT_DATE), $2::boolean
		FROM purchase_invoices pi
		JOIN purchase_orders po ON po.id = pi.purchase_order_id
		JOIN purchase_requests pr ON pr.id = po.purchase_request_id
		JOIN users ru ON ru.id = pr.requester_id
		JOIN departments d ON d.id = pr.department_id
		JOIN suppliers s ON s.id = po.supplier_id
		WHERE pi.id = $1
	`, invoiceID, canManage).Scan(invoiceBoardDestinations(&item)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return InvoiceBoardItem{}, ErrInvoiceNotFound
	}
	if err != nil {
		return InvoiceBoardItem{}, fmt.Errorf("load invoice item: %w", err)
	}
	return item, nil
}

func invoiceBoardDestinations(item *InvoiceBoardItem) []any {
	return []any{
		&item.PurchaseOrderID, &item.PurchaseRequestID, &item.RequestCode, &item.RequestTitle,
		&item.RequesterName, &item.DepartmentName, &item.SupplierID, &item.SupplierCode,
		&item.SupplierName, &item.OrderCode, &item.OrderStatus, &item.OrderAmount,
		&item.OrderCurrency, &item.ActualDeliveryOn, &item.InvoiceID, &item.InvoiceNumber,
		&item.IssuedOn, &item.DueOn, &item.InvoiceAmount, &item.InvoiceCurrency,
		&item.InvoiceStatus, &item.MatchStatus, &item.Note, &item.Version,
		&item.PaymentReference, &item.PaidOn, &item.InvoiceCreatedAt, &item.InvoiceUpdatedAt,
		&item.PaymentOverdue, &item.CanManage,
	}
}

func invoiceMatchStatus(orderStatus, invoiceAmount, invoiceCurrency, orderAmount, orderCurrency string) string {
	if orderStatus != "RECEIVED" {
		return "WAITING_RECEIPT"
	}
	if invoiceCurrency != orderCurrency {
		return "CURRENCY_MISMATCH"
	}
	if normalizedMoney(invoiceAmount) != normalizedMoney(orderAmount) {
		return "AMOUNT_MISMATCH"
	}
	return "MATCHED"
}

func invoiceEventType(action string) string {
	return map[string]string{
		"VERIFY": "INVOICE_VERIFIED", "DISPUTE": "INVOICE_DISPUTED",
		"REOPEN": "INVOICE_REOPENED", "MARK_PAID": "INVOICE_PAID",
	}[action]
}

func insertInvoiceEvent(
	ctx context.Context,
	tx pgx.Tx,
	invoiceID, eventType, fromStatus, toStatus, actorID string,
	actorRoles []string,
	comment, correlationID, idempotencyKey string,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO invoice_events (
			invoice_id, event_type, from_status, to_status, actor_id,
			actor_roles, comment, correlation_id, idempotency_key
		)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''))
	`, invoiceID, eventType, fromStatus, toStatus, actorID, actorRoles,
		comment, correlationID, idempotencyKey)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrIdempotencyConflict
		}
		return fmt.Errorf("insert invoice event: %w", err)
	}
	return nil
}

func nullableDate(value string) any {
	if value == "" {
		return nil
	}
	return value
}
