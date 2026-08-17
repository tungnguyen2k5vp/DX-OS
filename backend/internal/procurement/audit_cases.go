package procurement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/jackc/pgx/v5"
)

type AuditCase struct {
	ID           string    `json:"id"`
	CaseCode     string    `json:"caseCode"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Severity     string    `json:"severity"`
	Status       string    `json:"status"`
	ResourceType string    `json:"resourceType,omitempty"`
	ResourceID   *string   `json:"resourceId,omitempty"`
	OwnerUserID  *string   `json:"ownerUserId,omitempty"`
	OwnerName    string    `json:"ownerName,omitempty"`
	DueOn        *string   `json:"dueOn,omitempty"`
	Resolution   string    `json:"resolution,omitempty"`
	CreatedBy    string    `json:"createdBy"`
	Version      int64     `json:"version"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type AuditCaseList struct {
	Items     []AuditCase `json:"items"`
	Total     int         `json:"total"`
	CanManage bool        `json:"canManage"`
	CanExport bool        `json:"canExport"`
}

type AuditCaseInput struct {
	Title           string
	Description     string
	Severity        string
	ResourceType    string
	ResourceID      string
	OwnerUserID     string
	DueOn           string
	Status          string
	Resolution      string
	ExpectedVersion int64
	CorrelationID   string
}

type EvidenceAuditEvent struct {
	ResourceType  string          `json:"resourceType"`
	ResourceID    string          `json:"resourceId"`
	Action        string          `json:"action"`
	ActorName     string          `json:"actorName"`
	ActorRoles    []string        `json:"actorRoles"`
	FromStatus    *string         `json:"fromStatus"`
	ToStatus      *string         `json:"toStatus"`
	CorrelationID *string         `json:"correlationId"`
	Metadata      json.RawMessage `json:"metadata"`
	OccurredAt    time.Time       `json:"occurredAt"`
}

type EvidencePackage struct {
	GeneratedAt time.Time            `json:"generatedAt"`
	GeneratedBy string               `json:"generatedBy"`
	Request     PurchaseRequest      `json:"purchaseRequest"`
	Timeline    []TimelineEvent      `json:"timeline"`
	Attachments []Attachment         `json:"attachments"`
	Order       *PurchaseOrder       `json:"purchaseOrder,omitempty"`
	Receipts    []ReceiptRecord      `json:"receipts"`
	Invoices    []InvoiceBoardItem   `json:"invoices"`
	AuditEvents []EvidenceAuditEvent `json:"auditEvents"`
}

func validateAuditCase(input *AuditCaseInput, updating bool) error {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Severity = strings.ToUpper(strings.TrimSpace(input.Severity))
	input.Status = strings.ToUpper(strings.TrimSpace(input.Status))
	input.ResourceType = strings.TrimSpace(input.ResourceType)
	input.ResourceID = strings.TrimSpace(input.ResourceID)
	input.OwnerUserID = strings.TrimSpace(input.OwnerUserID)
	input.DueOn = strings.TrimSpace(input.DueOn)
	input.Resolution = strings.TrimSpace(input.Resolution)
	if input.Status == "" {
		input.Status = "OPEN"
	}
	var violations []FieldViolation
	if length := len([]rune(input.Title)); length < 3 || length > 255 {
		violations = append(violations, FieldViolation{Field: "title", Message: "must contain between 3 and 255 characters"})
	}
	if length := len([]rune(input.Description)); length < 10 || length > 10000 {
		violations = append(violations, FieldViolation{Field: "description", Message: "must contain between 10 and 10000 characters"})
	}
	if input.Severity != "LOW" && input.Severity != "MEDIUM" && input.Severity != "HIGH" && input.Severity != "CRITICAL" {
		violations = append(violations, FieldViolation{Field: "severity", Message: "must be LOW, MEDIUM, HIGH or CRITICAL"})
	}
	if input.Status != "OPEN" && input.Status != "IN_REMEDIATION" && input.Status != "RESOLVED" && input.Status != "CLOSED" {
		violations = append(violations, FieldViolation{Field: "status", Message: "must be OPEN, IN_REMEDIATION, RESOLVED or CLOSED"})
	}
	if input.ResourceID != "" && !uuidPatternForDomain.MatchString(input.ResourceID) {
		violations = append(violations, FieldViolation{Field: "resourceId", Message: "must be a valid UUID"})
	}
	if input.OwnerUserID != "" && !uuidPatternForDomain.MatchString(input.OwnerUserID) {
		violations = append(violations, FieldViolation{Field: "ownerUserId", Message: "must be a valid UUID"})
	}
	if input.DueOn != "" {
		if _, err := time.Parse(time.DateOnly, input.DueOn); err != nil {
			violations = append(violations, FieldViolation{Field: "dueOn", Message: "must use YYYY-MM-DD format"})
		}
	}
	if updating && input.ExpectedVersion < 1 {
		violations = append(violations, FieldViolation{Field: "expectedVersion", Message: "must be greater than zero"})
	}
	if (input.Status == "RESOLVED" || input.Status == "CLOSED") && len([]rune(input.Resolution)) < 5 {
		violations = append(violations, FieldViolation{Field: "resolution", Message: "is required when resolving or closing a case"})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func (s *Store) ListAuditCases(ctx context.Context, principal auth.Principal) (AuditCaseList, error) {
	if !hasRole(principal.Roles, "auditor") && !hasRole(principal.Roles, "dx_admin") {
		return AuditCaseList{}, ErrForbidden
	}
	user, err := s.ensureUser(ctx, principal)
	if err != nil {
		return AuditCaseList{}, err
	}
	rows, err := s.database.Query(ctx, `
		SELECT ac.id, ac.case_code, ac.title, ac.description, ac.severity, ac.status,
			COALESCE(ac.resource_type,''), ac.resource_id::text, ac.owner_user_id::text,
			COALESCE(ou.display_name,''), ac.due_on::text, COALESCE(ac.resolution,''),
			cu.display_name, ac.version, ac.created_at, ac.updated_at
		FROM audit_cases ac
		JOIN users cu ON cu.id=ac.created_by
		LEFT JOIN users ou ON ou.id=ac.owner_user_id
		WHERE ac.organization_id=$1
		ORDER BY CASE ac.status WHEN 'OPEN' THEN 1 WHEN 'IN_REMEDIATION' THEN 2 WHEN 'RESOLVED' THEN 3 ELSE 4 END,
			CASE ac.severity WHEN 'CRITICAL' THEN 1 WHEN 'HIGH' THEN 2 WHEN 'MEDIUM' THEN 3 ELSE 4 END,
			ac.due_on NULLS LAST, ac.updated_at DESC
	`, user.OrganizationID)
	if err != nil {
		return AuditCaseList{}, fmt.Errorf("list audit cases: %w", err)
	}
	defer rows.Close()
	result := AuditCaseList{Items: make([]AuditCase, 0), CanManage: hasRole(principal.Roles, "auditor"), CanExport: hasRole(principal.Roles, "auditor")}
	for rows.Next() {
		var item AuditCase
		if err = rows.Scan(&item.ID, &item.CaseCode, &item.Title, &item.Description, &item.Severity, &item.Status, &item.ResourceType, &item.ResourceID, &item.OwnerUserID, &item.OwnerName, &item.DueOn, &item.Resolution, &item.CreatedBy, &item.Version, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return AuditCaseList{}, fmt.Errorf("scan audit case: %w", err)
		}
		result.Items = append(result.Items, item)
	}
	if err = rows.Err(); err != nil {
		return AuditCaseList{}, fmt.Errorf("iterate audit cases: %w", err)
	}
	result.Total = len(result.Items)
	return result, nil
}

func (s *Store) CreateAuditCase(ctx context.Context, principal auth.Principal, input AuditCaseInput) (AuditCase, error) {
	if !hasRole(principal.Roles, "auditor") {
		return AuditCase{}, ErrForbidden
	}
	if err := validateAuditCase(&input, false); err != nil {
		return AuditCase{}, err
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return AuditCase{}, fmt.Errorf("begin audit case: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return AuditCase{}, err
	}
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO audit_cases (organization_id, case_code, title, description, severity, status, resource_type, resource_id, owner_user_id, due_on, resolution, created_by)
		VALUES ($1, 'AC-'||to_char(CURRENT_DATE,'YYYY')||'-'||lpad(nextval('audit_case_code_seq')::text,6,'0'), $2,$3,$4,$5,NULLIF($6,''),NULLIF($7,'')::uuid,NULLIF($8,'')::uuid,NULLIF($9,'')::date,NULLIF($10,''),$11)
		RETURNING id
	`, user.OrganizationID, input.Title, input.Description, input.Severity, input.Status, input.ResourceType, input.ResourceID, input.OwnerUserID, input.DueOn, input.Resolution, user.ID).Scan(&id)
	if err != nil {
		return AuditCase{}, fmt.Errorf("insert audit case: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_case_events (audit_case_id,event_type,to_status,actor_id,comment,correlation_id) VALUES ($1,'CASE_CREATED',$2,$3,$4,NULLIF($5,''))`, id, input.Status, user.ID, input.Description, input.CorrelationID)
	if err != nil {
		return AuditCase{}, fmt.Errorf("insert audit case event: %w", err)
	}
	if err = insertResourceAudit(ctx, tx, "audit_case", id, "AUDIT_CASE_CREATED", user.ID, principal.Roles, "", input.Status, input.CorrelationID); err != nil {
		return AuditCase{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return AuditCase{}, fmt.Errorf("commit audit case: %w", err)
	}
	return s.getAuditCase(ctx, user.OrganizationID, id)
}

func (s *Store) UpdateAuditCase(ctx context.Context, principal auth.Principal, caseID string, input AuditCaseInput) (AuditCase, error) {
	if !hasRole(principal.Roles, "auditor") {
		return AuditCase{}, ErrForbidden
	}
	if err := validateAuditCase(&input, true); err != nil {
		return AuditCase{}, err
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return AuditCase{}, fmt.Errorf("begin update audit case: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return AuditCase{}, err
	}
	var currentStatus string
	var version int64
	err = tx.QueryRow(ctx, `SELECT status,version FROM audit_cases WHERE id=$1 AND organization_id=$2 FOR UPDATE`, caseID, user.OrganizationID).Scan(&currentStatus, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return AuditCase{}, ErrAuditCaseNotFound
	}
	if err != nil {
		return AuditCase{}, fmt.Errorf("lock audit case: %w", err)
	}
	if version != input.ExpectedVersion {
		return AuditCase{}, ErrAuditCaseVersion
	}
	_, err = tx.Exec(ctx, `UPDATE audit_cases SET title=$2,description=$3,severity=$4,status=$5,resource_type=NULLIF($6,''),resource_id=NULLIF($7,'')::uuid,owner_user_id=NULLIF($8,'')::uuid,due_on=NULLIF($9,'')::date,resolution=NULLIF($10,''),version=version+1,updated_at=now() WHERE id=$1`, caseID, input.Title, input.Description, input.Severity, input.Status, input.ResourceType, input.ResourceID, input.OwnerUserID, input.DueOn, input.Resolution)
	if err != nil {
		return AuditCase{}, fmt.Errorf("update audit case: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_case_events (audit_case_id,event_type,from_status,to_status,actor_id,comment,correlation_id) VALUES ($1,'CASE_UPDATED',$2,$3,$4,NULLIF($5,''),NULLIF($6,''))`, caseID, currentStatus, input.Status, user.ID, input.Resolution, input.CorrelationID)
	if err != nil {
		return AuditCase{}, fmt.Errorf("insert audit case update event: %w", err)
	}
	if err = insertResourceAudit(ctx, tx, "audit_case", caseID, "AUDIT_CASE_UPDATED", user.ID, principal.Roles, currentStatus, input.Status, input.CorrelationID); err != nil {
		return AuditCase{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return AuditCase{}, fmt.Errorf("commit audit case update: %w", err)
	}
	return s.getAuditCase(ctx, user.OrganizationID, caseID)
}

func (s *Store) getAuditCase(ctx context.Context, organizationID, caseID string) (AuditCase, error) {
	var item AuditCase
	err := s.database.QueryRow(ctx, `SELECT ac.id,ac.case_code,ac.title,ac.description,ac.severity,ac.status,COALESCE(ac.resource_type,''),ac.resource_id::text,ac.owner_user_id::text,COALESCE(ou.display_name,''),ac.due_on::text,COALESCE(ac.resolution,''),cu.display_name,ac.version,ac.created_at,ac.updated_at FROM audit_cases ac JOIN users cu ON cu.id=ac.created_by LEFT JOIN users ou ON ou.id=ac.owner_user_id WHERE ac.id=$1 AND ac.organization_id=$2`, caseID, organizationID).Scan(&item.ID, &item.CaseCode, &item.Title, &item.Description, &item.Severity, &item.Status, &item.ResourceType, &item.ResourceID, &item.OwnerUserID, &item.OwnerName, &item.DueOn, &item.Resolution, &item.CreatedBy, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return AuditCase{}, ErrAuditCaseNotFound
	}
	if err != nil {
		return AuditCase{}, fmt.Errorf("get audit case: %w", err)
	}
	return item, nil
}

func (s *Store) EvidencePackage(ctx context.Context, principal auth.Principal, requestID string) (EvidencePackage, error) {
	if !hasRole(principal.Roles, "auditor") {
		return EvidencePackage{}, ErrForbidden
	}
	request, err := s.Get(ctx, principal, requestID)
	if err != nil {
		return EvidencePackage{}, err
	}
	timeline, err := s.Timeline(ctx, principal, requestID, TimelineInput{Page: 1, PageSize: 100})
	if err != nil {
		return EvidencePackage{}, err
	}
	attachments, err := s.ListAttachments(ctx, principal, requestID)
	if err != nil {
		return EvidencePackage{}, err
	}
	receipts, err := s.ListReceipts(ctx, principal, requestID)
	if err != nil {
		return EvidencePackage{}, err
	}
	result := EvidencePackage{GeneratedAt: time.Now().UTC(), GeneratedBy: principal.Username, Request: request, Timeline: timeline.Items, Attachments: attachments.Items, Receipts: receipts.Items, Invoices: make([]InvoiceBoardItem, 0), AuditEvents: make([]EvidenceAuditEvent, 0)}
	if order, loadErr := s.loadPurchaseOrder(ctx, requestID); loadErr == nil {
		result.Order = &order
	}
	rows, err := s.database.Query(ctx, `SELECT pi.id FROM purchase_invoices pi JOIN purchase_orders po ON po.id=pi.purchase_order_id WHERE po.purchase_request_id=$1 ORDER BY pi.created_at`, requestID)
	if err != nil {
		return EvidencePackage{}, fmt.Errorf("list evidence invoices: %w", err)
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return EvidencePackage{}, err
		}
		invoice, loadErr := s.loadInvoiceItem(ctx, id, false)
		if loadErr != nil {
			rows.Close()
			return EvidencePackage{}, loadErr
		}
		result.Invoices = append(result.Invoices, invoice)
	}
	rows.Close()
	auditRows, err := s.database.Query(ctx, `SELECT al.resource_type,al.resource_id::text,al.action,u.display_name,al.actor_roles,al.from_status,al.to_status,al.correlation_id,al.metadata,al.occurred_at FROM audit_logs al JOIN users u ON u.id=al.actor_id WHERE (al.resource_type='purchase_request' AND al.resource_id=$1) OR al.resource_id IN (SELECT po.id FROM purchase_orders po WHERE po.purchase_request_id=$1) OR al.resource_id IN (SELECT pi.id FROM purchase_invoices pi JOIN purchase_orders po ON po.id=pi.purchase_order_id WHERE po.purchase_request_id=$1) ORDER BY al.occurred_at,al.id`, requestID)
	if err != nil {
		return EvidencePackage{}, fmt.Errorf("list evidence audit: %w", err)
	}
	defer auditRows.Close()
	for auditRows.Next() {
		var event EvidenceAuditEvent
		if err = auditRows.Scan(&event.ResourceType, &event.ResourceID, &event.Action, &event.ActorName, &event.ActorRoles, &event.FromStatus, &event.ToStatus, &event.CorrelationID, &event.Metadata, &event.OccurredAt); err != nil {
			return EvidencePackage{}, err
		}
		result.AuditEvents = append(result.AuditEvents, event)
	}
	return result, auditRows.Err()
}
