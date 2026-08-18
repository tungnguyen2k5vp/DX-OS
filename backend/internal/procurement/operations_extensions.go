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

type ReceiptItemInput struct {
	PurchaseRequestItemID string
	QuantityReceived      string
	Condition             string
	Note                  string
}

type RecordReceiptInput struct {
	ExpectedVersion int64
	Outcome         string
	ReceivedOn      string
	Note            string
	Items           []ReceiptItemInput
	IdempotencyKey  string
	CorrelationID   string
}

type ReceiptItem struct {
	PurchaseRequestItemID string `json:"purchaseRequestItemId"`
	LineNumber            int    `json:"lineNumber"`
	Description           string `json:"description"`
	OrderedQuantity       string `json:"orderedQuantity"`
	QuantityReceived      string `json:"quantityReceived"`
	Condition             string `json:"condition"`
	Note                  string `json:"note,omitempty"`
}

type ReceiptRecord struct {
	ID            string        `json:"id"`
	ReceiptNumber string        `json:"receiptNumber"`
	Outcome       string        `json:"outcome"`
	ReceivedOn    string        `json:"receivedOn"`
	Note          string        `json:"note"`
	CreatedBy     string        `json:"createdBy"`
	CreatedAt     time.Time     `json:"createdAt"`
	Items         []ReceiptItem `json:"items"`
}

type ReceiptHistory struct {
	Items []ReceiptRecord `json:"items"`
	Total int             `json:"total"`
}

type UpdatePurchaseOrderInput struct {
	SupplierID         string
	ExternalReference  string
	ExpectedDeliveryOn string
	Note               string
	ExpectedVersion    int64
	CorrelationID      string
}

type CancelPurchaseOrderInput struct {
	ExpectedVersion int64
	Reason          string
	IdempotencyKey  string
	CorrelationID   string
}

func ValidateRecordReceipt(input *RecordReceiptInput) error {
	input.Outcome = strings.ToUpper(strings.TrimSpace(input.Outcome))
	input.ReceivedOn = strings.TrimSpace(input.ReceivedOn)
	input.Note = strings.TrimSpace(input.Note)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	var violations []FieldViolation
	if input.ExpectedVersion <= 0 {
		violations = append(violations, FieldViolation{Field: "expectedVersion", Message: "Phải lớn hơn 0."})
	}
	validOutcome := map[string]bool{"PARTIAL": true, "COMPLETE": true, "DAMAGED": true, "WRONG_ITEM": true, "REJECTED": true}
	if !validOutcome[input.Outcome] {
		violations = append(violations, FieldViolation{Field: "outcome", Message: "Phải là PARTIAL (nhận một phần), COMPLETE (nhận đủ), DAMAGED (hư hỏng), WRONG_ITEM (sai hàng) hoặc REJECTED (từ chối nhận)."})
	}
	if _, err := time.Parse("2006-01-02", input.ReceivedOn); err != nil {
		violations = append(violations, FieldViolation{Field: "receivedOn", Message: "Phải có định dạng YYYY-MM-DD."})
	}
	if length := len([]rune(input.Note)); length < 5 || length > 5000 {
		violations = append(violations, FieldViolation{Field: "note", Message: "Phải có từ 5 đến 5.000 ký tự."})
	}
	if !idempotencyPattern.MatchString(input.IdempotencyKey) {
		violations = append(violations, FieldViolation{Field: "Idempotency-Key", Message: "Phải có từ 8 đến 255 ký tự an toàn."})
	}
	if len(input.Items) == 0 || len(input.Items) > 100 {
		violations = append(violations, FieldViolation{Field: "items", Message: "Phải có từ 1 đến 100 dòng giao nhận."})
	}
	seen := map[string]bool{}
	for index := range input.Items {
		item := &input.Items[index]
		item.PurchaseRequestItemID = strings.TrimSpace(item.PurchaseRequestItemID)
		item.QuantityReceived = strings.TrimSpace(item.QuantityReceived)
		item.Condition = strings.ToUpper(strings.TrimSpace(item.Condition))
		item.Note = strings.TrimSpace(item.Note)
		prefix := fmt.Sprintf("items[%d]", index)
		if !uuidPatternForDomain.MatchString(item.PurchaseRequestItemID) || seen[item.PurchaseRequestItemID] {
			violations = append(violations, FieldViolation{Field: prefix + ".purchaseRequestItemId", Message: "Phải là UUID hợp lệ và không trùng lặp."})
		}
		seen[item.PurchaseRequestItemID] = true
		if !quantityPattern.MatchString(item.QuantityReceived) {
			violations = append(violations, FieldViolation{Field: prefix + ".quantityReceived", Message: "Phải là số lượng không âm và có tối đa 4 chữ số thập phân."})
		}
		if item.Condition != "ACCEPTED" && item.Condition != "DAMAGED" && item.Condition != "WRONG_ITEM" && item.Condition != "REJECTED" {
			violations = append(violations, FieldViolation{Field: prefix + ".condition", Message: "Phải là ACCEPTED (đã nhận), DAMAGED (hư hỏng), WRONG_ITEM (sai hàng) hoặc REJECTED (từ chối nhận)."})
		}
		if len([]rune(item.Note)) > 1000 {
			violations = append(violations, FieldViolation{Field: prefix + ".note", Message: "Không được vượt quá 1.000 ký tự."})
		}
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func (s *Store) RecordReceipt(ctx context.Context, principal auth.Principal, requestID string, input RecordReceiptInput) (PurchaseOrder, error) {
	if hasRole(principal.Roles, "auditor") {
		return PurchaseOrder{}, ErrForbidden
	}
	if err := ValidateRecordReceipt(&input); err != nil {
		return PurchaseOrder{}, err
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("begin record receipt: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return PurchaseOrder{}, err
	}
	var orderID, status, requesterID, departmentID, requestCode, requestTitle string
	var version int64
	err = tx.QueryRow(ctx, `
		SELECT po.id, po.status, po.version, pr.requester_id, pr.department_id,
			pr.request_code, pr.title
		FROM purchase_orders po
		JOIN purchase_requests pr ON pr.id = po.purchase_request_id
		WHERE po.purchase_request_id = $1 AND po.organization_id = $2
		FOR UPDATE OF po
	`, requestID, user.OrganizationID).Scan(&orderID, &status, &version, &requesterID, &departmentID, &requestCode, &requestTitle)
	if errors.Is(err, pgx.ErrNoRows) {
		return PurchaseOrder{}, ErrPurchaseOrderNotFound
	}
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("lock receipt order: %w", err)
	}
	canReceive := (requesterID == user.ID && (hasRole(principal.Roles, "employee") || hasRole(principal.Roles, "department_manager"))) ||
		(hasRole(principal.Roles, "department_manager") && departmentID == user.DepartmentID)
	if !canReceive {
		return PurchaseOrder{}, ErrForbidden
	}
	var replayOrderID string
	err = tx.QueryRow(ctx, `SELECT purchase_order_id FROM purchase_order_receipts WHERE idempotency_key = $1`, input.IdempotencyKey).Scan(&replayOrderID)
	if err == nil {
		if replayOrderID != orderID {
			return PurchaseOrder{}, ErrIdempotencyConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return PurchaseOrder{}, fmt.Errorf("commit receipt replay: %w", err)
		}
		return s.loadPurchaseOrder(ctx, requestID)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return PurchaseOrder{}, fmt.Errorf("check receipt idempotency: %w", err)
	}
	if status != "ORDERED" && status != "PARTIALLY_RECEIVED" && status != "RECEIPT_EXCEPTION" {
		return PurchaseOrder{}, ErrInvalidFulfillment
	}
	if version != input.ExpectedVersion {
		return PurchaseOrder{}, ErrVersionConflict
	}
	var receiptID string
	err = tx.QueryRow(ctx, `
		INSERT INTO purchase_order_receipts (
			purchase_order_id, outcome, received_on, note, created_by, correlation_id, idempotency_key
		) VALUES ($1, $2, $3::date, $4, $5, NULLIF($6, ''), $7)
		RETURNING id
	`, orderID, input.Outcome, input.ReceivedOn, input.Note, user.ID, input.CorrelationID, input.IdempotencyKey).Scan(&receiptID)
	if err != nil {
		if isUniqueViolation(err) {
			return PurchaseOrder{}, ErrIdempotencyConflict
		}
		return PurchaseOrder{}, fmt.Errorf("insert receipt: %w", err)
	}
	for _, item := range input.Items {
		var ordered, alreadyReceived string
		err = tx.QueryRow(ctx, `
			SELECT pri.quantity::text,
				COALESCE((SELECT sum(pori.quantity_received)
					FROM purchase_order_receipt_items pori
					JOIN purchase_order_receipts por ON por.id = pori.receipt_id
					WHERE por.purchase_order_id = $2 AND pori.purchase_request_item_id = pri.id
						AND pori.condition = 'ACCEPTED' AND por.id <> $3), 0)::text
			FROM purchase_request_items pri
			WHERE pri.id = $1 AND pri.purchase_request_id = $4
		`, item.PurchaseRequestItemID, orderID, receiptID, requestID).Scan(&ordered, &alreadyReceived)
		if errors.Is(err, pgx.ErrNoRows) {
			return PurchaseOrder{}, &ValidationError{Violations: []FieldViolation{{Field: "items", Message: "Có dòng hàng không thuộc đơn hàng này."}}}
		}
		if err != nil {
			return PurchaseOrder{}, fmt.Errorf("load receipt line: %w", err)
		}
		if item.Condition == "ACCEPTED" && addRat(alreadyReceived, item.QuantityReceived).Cmp(mustRat(ordered)) > 0 {
			return PurchaseOrder{}, &ValidationError{Violations: []FieldViolation{{Field: "items", Message: "Số lượng nhận vượt quá số lượng đã đặt."}}}
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO purchase_order_receipt_items (
				receipt_id, purchase_request_item_id, quantity_received, condition, note
			) VALUES ($1, $2, $3::numeric, $4, NULLIF($5, ''))
		`, receiptID, item.PurchaseRequestItemID, item.QuantityReceived, item.Condition, item.Note)
		if err != nil {
			return PurchaseOrder{}, fmt.Errorf("insert receipt line: %w", err)
		}
	}
	var allReceived bool
	err = tx.QueryRow(ctx, `
		SELECT bool_and(COALESCE(received.quantity_received, 0) >= pri.quantity)
		FROM purchase_request_items pri
		LEFT JOIN (
			SELECT pori.purchase_request_item_id, sum(pori.quantity_received) AS quantity_received
			FROM purchase_order_receipt_items pori
			JOIN purchase_order_receipts por ON por.id = pori.receipt_id
			WHERE por.purchase_order_id = $1 AND pori.condition = 'ACCEPTED'
			GROUP BY pori.purchase_request_item_id
		) received ON received.purchase_request_item_id = pri.id
		WHERE pri.purchase_request_id = $2
	`, orderID, requestID).Scan(&allReceived)
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("reconcile receipt quantities: %w", err)
	}
	if input.Outcome == "COMPLETE" && !allReceived {
		return PurchaseOrder{}, &ValidationError{Violations: []FieldViolation{{Field: "outcome", Message: "Trạng thái COMPLETE yêu cầu mọi dòng hàng đã được nhận đủ."}}}
	}
	toStatus := "PARTIALLY_RECEIVED"
	if allReceived {
		toStatus = "RECEIVED"
	} else if input.Outcome == "DAMAGED" || input.Outcome == "WRONG_ITEM" || input.Outcome == "REJECTED" {
		toStatus = "RECEIPT_EXCEPTION"
	}
	_, err = tx.Exec(ctx, `
		UPDATE purchase_orders
		SET status = $2, actual_delivery_on = $3::date, received_by = $4,
			received_at = now(), version = version + 1, updated_at = now()
		WHERE id = $1
	`, orderID, toStatus, input.ReceivedOn, user.ID)
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("update receipt order: %w", err)
	}
	eventType := map[string]string{
		"PARTIAL": "DELIVERY_PARTIALLY_RECEIVED", "COMPLETE": "DELIVERY_RECEIVED",
		"DAMAGED": "DELIVERY_DAMAGED", "WRONG_ITEM": "DELIVERY_WRONG_ITEM", "REJECTED": "DELIVERY_REJECTED",
	}[input.Outcome]
	_, err = tx.Exec(ctx, `
		INSERT INTO process_events (
			purchase_request_id, event_type, from_status, to_status, actor_id,
			actor_roles, comment, correlation_id, idempotency_key,
			metadata
		) VALUES ($1, $2, 'APPROVED', 'APPROVED', $3, $4, $5, NULLIF($6, ''), $7,
			jsonb_build_object('purchaseOrderStatus', $8::text, 'receiptId', $9::text))
	`, requestID, eventType, user.ID, principal.Roles, input.Note, input.CorrelationID, input.IdempotencyKey, toStatus, receiptID)
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("insert receipt event: %w", err)
	}
	if err = insertAudit(ctx, tx, requestID, eventType, user.ID, principal.Roles, string(StatusApproved), StatusApproved, input.CorrelationID); err != nil {
		return PurchaseOrder{}, err
	}
	if err = insertResourceAudit(ctx, tx, "purchase_order", orderID, eventType, user.ID, principal.Roles, status, toStatus, input.CorrelationID); err != nil {
		return PurchaseOrder{}, err
	}
	for _, recipient := range []notifications.QueueInput{
		{EventType: eventType, ResourceType: "purchase_request", ResourceID: requestID, OrganizationID: user.OrganizationID, DepartmentID: departmentID, RecipientRole: "finance", ActorID: user.ID, Title: "Cập nhật giao nhận " + requestCode, Body: requestTitle + " - " + input.Note},
		{EventType: eventType, ResourceType: "purchase_request", ResourceID: requestID, OrganizationID: user.OrganizationID, DepartmentID: departmentID, RecipientUserID: requesterID, ActorID: user.ID, Title: "Cập nhật giao nhận " + requestCode, Body: requestTitle + " - " + input.Note},
	} {
		if err = notifications.Queue(ctx, tx, recipient); err != nil {
			return PurchaseOrder{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return PurchaseOrder{}, fmt.Errorf("commit receipt: %w", err)
	}
	return s.loadPurchaseOrder(ctx, requestID)
}

func (s *Store) ListReceipts(ctx context.Context, principal auth.Principal, requestID string) (ReceiptHistory, error) {
	if _, err := s.Get(ctx, principal, requestID); err != nil {
		return ReceiptHistory{}, err
	}
	rows, err := s.database.Query(ctx, `
		SELECT por.id, 'GR-' || lpad(por.receipt_number::text, 6, '0'), por.outcome,
			por.received_on::text, por.note, u.display_name, por.created_at
		FROM purchase_order_receipts por
		JOIN purchase_orders po ON po.id = por.purchase_order_id
		JOIN users u ON u.id = por.created_by
		WHERE po.purchase_request_id = $1
		ORDER BY por.created_at DESC, por.id DESC
	`, requestID)
	if err != nil {
		return ReceiptHistory{}, fmt.Errorf("list receipts: %w", err)
	}
	defer rows.Close()
	result := ReceiptHistory{Items: make([]ReceiptRecord, 0)}
	for rows.Next() {
		var receipt ReceiptRecord
		if err = rows.Scan(&receipt.ID, &receipt.ReceiptNumber, &receipt.Outcome, &receipt.ReceivedOn, &receipt.Note, &receipt.CreatedBy, &receipt.CreatedAt); err != nil {
			return ReceiptHistory{}, fmt.Errorf("scan receipt: %w", err)
		}
		receipt.Items, err = s.listReceiptItems(ctx, receipt.ID)
		if err != nil {
			return ReceiptHistory{}, err
		}
		result.Items = append(result.Items, receipt)
	}
	if err = rows.Err(); err != nil {
		return ReceiptHistory{}, fmt.Errorf("iterate receipts: %w", err)
	}
	result.Total = len(result.Items)
	return result, nil
}

func (s *Store) listReceiptItems(ctx context.Context, receiptID string) ([]ReceiptItem, error) {
	rows, err := s.database.Query(ctx, `
		SELECT pri.id, pri.line_number, pri.description, pri.quantity::text,
			pori.quantity_received::text, pori.condition, COALESCE(pori.note, '')
		FROM purchase_order_receipt_items pori
		JOIN purchase_request_items pri ON pri.id = pori.purchase_request_item_id
		WHERE pori.receipt_id = $1 ORDER BY pri.line_number
	`, receiptID)
	if err != nil {
		return nil, fmt.Errorf("list receipt items: %w", err)
	}
	defer rows.Close()
	items := make([]ReceiptItem, 0)
	for rows.Next() {
		var item ReceiptItem
		if err = rows.Scan(&item.PurchaseRequestItemID, &item.LineNumber, &item.Description, &item.OrderedQuantity, &item.QuantityReceived, &item.Condition, &item.Note); err != nil {
			return nil, fmt.Errorf("scan receipt item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) UpdatePurchaseOrder(ctx context.Context, principal auth.Principal, requestID string, input UpdatePurchaseOrderInput) (PurchaseOrder, error) {
	if !hasRole(principal.Roles, "finance") || hasRole(principal.Roles, "auditor") {
		return PurchaseOrder{}, ErrForbidden
	}
	createInput := CreatePurchaseOrderInput{PurchaseRequestID: requestID, SupplierID: input.SupplierID, ExternalReference: input.ExternalReference, ExpectedDeliveryOn: input.ExpectedDeliveryOn, Note: input.Note, IdempotencyKey: "update-po-placeholder"}
	if err := ValidateCreatePurchaseOrder(&createInput); err != nil {
		return PurchaseOrder{}, err
	}
	if input.ExpectedVersion <= 0 {
		return PurchaseOrder{}, &ValidationError{Violations: []FieldViolation{{Field: "expectedVersion", Message: "Phải lớn hơn 0."}}}
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("begin update order: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return PurchaseOrder{}, err
	}
	var orderID, status string
	var version int64
	err = tx.QueryRow(ctx, `SELECT id, status, version FROM purchase_orders WHERE purchase_request_id = $1 AND organization_id = $2 FOR UPDATE`, requestID, user.OrganizationID).Scan(&orderID, &status, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return PurchaseOrder{}, ErrPurchaseOrderNotFound
	}
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("lock order update: %w", err)
	}
	if status != "ORDERED" || version != input.ExpectedVersion {
		if version != input.ExpectedVersion {
			return PurchaseOrder{}, ErrVersionConflict
		}
		return PurchaseOrder{}, ErrInvalidFulfillment
	}
	var supplierActive bool
	err = tx.QueryRow(ctx, `SELECT status = 'ACTIVE' FROM suppliers WHERE id = $1 AND organization_id = $2`, input.SupplierID, user.OrganizationID).Scan(&supplierActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return PurchaseOrder{}, ErrSupplierNotFound
	}
	if err != nil || !supplierActive {
		return PurchaseOrder{}, ErrInvalidFulfillment
	}
	_, err = tx.Exec(ctx, `UPDATE purchase_orders SET supplier_id=$2, external_reference=NULLIF($3,''), expected_delivery_on=$4::date, note=NULLIF($5,''), version=version+1, updated_at=now() WHERE id=$1`, orderID, input.SupplierID, input.ExternalReference, input.ExpectedDeliveryOn, input.Note)
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("update order: %w", err)
	}
	if err = insertResourceAudit(ctx, tx, "purchase_order", orderID, "PURCHASE_ORDER_UPDATED", user.ID, principal.Roles, status, status, input.CorrelationID); err != nil {
		return PurchaseOrder{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return PurchaseOrder{}, fmt.Errorf("commit update order: %w", err)
	}
	return s.loadPurchaseOrder(ctx, requestID)
}

func (s *Store) CancelPurchaseOrder(ctx context.Context, principal auth.Principal, requestID string, input CancelPurchaseOrderInput) (PurchaseOrder, error) {
	if !hasRole(principal.Roles, "finance") || hasRole(principal.Roles, "auditor") {
		return PurchaseOrder{}, ErrForbidden
	}
	input.Reason = strings.TrimSpace(input.Reason)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.ExpectedVersion <= 0 || len([]rune(input.Reason)) < 10 || len([]rune(input.Reason)) > 2000 || !idempotencyPattern.MatchString(input.IdempotencyKey) {
		return PurchaseOrder{}, &ValidationError{Violations: []FieldViolation{{Field: "reason", Message: "Lý do phải có từ 10 đến 2.000 ký tự và cần Idempotency-Key hợp lệ."}}}
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("begin cancel order: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return PurchaseOrder{}, err
	}
	var orderID, status, requestStatus string
	var version int64
	err = tx.QueryRow(ctx, `SELECT po.id,po.status,po.version,pr.status FROM purchase_orders po JOIN purchase_requests pr ON pr.id=po.purchase_request_id WHERE po.purchase_request_id=$1 AND po.organization_id=$2 FOR UPDATE OF po`, requestID, user.OrganizationID).Scan(&orderID, &status, &version, &requestStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return PurchaseOrder{}, ErrPurchaseOrderNotFound
	}
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("lock cancel order: %w", err)
	}
	var replayRequestID, replayType string
	err = tx.QueryRow(ctx, `SELECT purchase_request_id,event_type FROM process_events WHERE idempotency_key=$1`, input.IdempotencyKey).Scan(&replayRequestID, &replayType)
	if err == nil {
		if replayRequestID != requestID || replayType != "PURCHASE_ORDER_CANCELLED" {
			return PurchaseOrder{}, ErrIdempotencyConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return PurchaseOrder{}, err
		}
		return s.loadPurchaseOrder(ctx, requestID)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return PurchaseOrder{}, fmt.Errorf("check cancel idempotency: %w", err)
	}
	if version != input.ExpectedVersion {
		return PurchaseOrder{}, ErrVersionConflict
	}
	if status != "ORDERED" {
		return PurchaseOrder{}, ErrInvalidFulfillment
	}
	var dependentCount int
	if err = tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM purchase_order_receipts WHERE purchase_order_id=$1) + (SELECT count(*) FROM purchase_invoices WHERE purchase_order_id=$1)`, orderID).Scan(&dependentCount); err != nil {
		return PurchaseOrder{}, fmt.Errorf("check cancel dependencies: %w", err)
	}
	if dependentCount > 0 {
		return PurchaseOrder{}, ErrInvalidFulfillment
	}
	_, err = tx.Exec(ctx, `UPDATE purchase_orders SET status='CANCELLED', cancelled_by=$2, cancelled_at=now(), cancellation_reason=$3, version=version+1, updated_at=now() WHERE id=$1`, orderID, user.ID, input.Reason)
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("cancel order: %w", err)
	}
	if err = insertResourceAudit(ctx, tx, "purchase_order", orderID, "PURCHASE_ORDER_CANCELLED", user.ID, principal.Roles, status, "CANCELLED", input.CorrelationID); err != nil {
		return PurchaseOrder{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO process_events(purchase_request_id,event_type,from_status,to_status,actor_id,actor_roles,comment,correlation_id,idempotency_key,metadata) VALUES($1,'PURCHASE_ORDER_CANCELLED',$2,$2,$3,$4,$5,NULLIF($6,''),$7,jsonb_build_object('purchaseOrderStatus','CANCELLED'))`, requestID, requestStatus, user.ID, principal.Roles, input.Reason, input.CorrelationID, input.IdempotencyKey)
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("insert cancel order event: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return PurchaseOrder{}, fmt.Errorf("commit cancel order: %w", err)
	}
	return s.loadPurchaseOrder(ctx, requestID)
}

func mustRat(value string) *big.Rat {
	result, ok := new(big.Rat).SetString(value)
	if !ok {
		return new(big.Rat)
	}
	return result
}

func addRat(left, right string) *big.Rat {
	return new(big.Rat).Add(mustRat(left), mustRat(right))
}
