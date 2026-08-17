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
	"github.com/dx-os-lab/dx-os/backend/internal/platform/identity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultDepartmentCode = "GENERAL"

type userProfile = identity.Profile

type Store struct {
	database  *pgxpool.Pool
	documents documentStore
}

func NewStore(database *pgxpool.Pool, documents documentStore) *Store {
	return &Store{database: database, documents: documents}
}

func (s *Store) Create(
	ctx context.Context,
	principal auth.Principal,
	input CreateInput,
	correlationID string,
) (PurchaseRequest, error) {
	if !CanCreate(principal) {
		return PurchaseRequest{}, ErrForbidden
	}
	if err := ValidateCreate(&input); err != nil {
		return PurchaseRequest{}, err
	}

	tx, err := s.database.Begin(ctx)
	if err != nil {
		return PurchaseRequest{}, fmt.Errorf("begin create purchase request: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return PurchaseRequest{}, err
	}

	var requestID string
	err = tx.QueryRow(ctx, `
		INSERT INTO purchase_requests (
			request_code,
			requester_id,
			department_id,
			title,
			reason,
			currency,
			cost_center
		)
		VALUES (
			'PR-' || to_char(CURRENT_DATE, 'YYYY') || '-' ||
				lpad(nextval('purchase_request_code_seq')::text, 6, '0'),
			$1, $2, $3, $4, $5, $6
		)
		RETURNING id
	`, user.ID, user.DepartmentID, input.Title, input.Reason, input.Currency, input.CostCenter).Scan(&requestID)
	if err != nil {
		return PurchaseRequest{}, fmt.Errorf("insert purchase request: %w", err)
	}

	for index, item := range input.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO purchase_request_items (
				purchase_request_id,
				line_number,
				description,
				quantity,
				unit,
				unit_price
			)
			VALUES ($1, $2, $3, $4::numeric, $5, $6::numeric)
		`, requestID, index+1, item.Description, item.Quantity, item.Unit, item.UnitPrice)
		if err != nil {
			return PurchaseRequest{}, fmt.Errorf("insert purchase request item %d: %w", index+1, err)
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE purchase_requests
		SET total_amount = (
				SELECT COALESCE(SUM(line_total), 0)
				FROM purchase_request_items
				WHERE purchase_request_id = $1
			),
			updated_at = now()
		WHERE id = $1
	`, requestID)
	if err != nil {
		return PurchaseRequest{}, fmt.Errorf("calculate purchase request total: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO process_events (
			purchase_request_id,
			event_type,
			to_status,
			actor_id,
			actor_roles,
			correlation_id
		)
		VALUES ($1, 'DRAFT_CREATED', 'DRAFT', $2, $3, NULLIF($4, ''))
	`, requestID, user.ID, principal.Roles, correlationID)
	if err != nil {
		return PurchaseRequest{}, fmt.Errorf("insert draft-created event: %w", err)
	}
	if err = insertAudit(
		ctx, tx, requestID, "DRAFT_CREATED", user.ID, principal.Roles,
		"", StatusDraft, correlationID,
	); err != nil {
		return PurchaseRequest{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return PurchaseRequest{}, fmt.Errorf("commit purchase request: %w", err)
	}

	return s.getUnscoped(ctx, requestID)
}

func (s *Store) List(
	ctx context.Context,
	principal auth.Principal,
	input ListInput,
) (ListResult, error) {
	scope, err := ScopeFor(principal)
	if err != nil {
		return ListResult{}, err
	}
	user, err := s.ensureUser(ctx, principal)
	if err != nil {
		return ListResult{}, err
	}

	var conditions []string
	args := []any{}
	addArgument := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	switch scope {
	case ScopeOwn:
		conditions = append(conditions, "pr.requester_id = "+addArgument(user.ID))
	case ScopeDepartment:
		conditions = append(conditions, "pr.department_id = "+addArgument(user.DepartmentID))
	case ScopeFinance:
		conditions = append(conditions, "d.organization_id = "+addArgument(user.OrganizationID))
		conditions = append(conditions, "pr.status IN ('MANAGER_APPROVED', 'APPROVED', 'REJECTED')")
	case ScopeAll:
		// Auditor scope is read-only across the application database.
	}
	if input.Status != nil {
		conditions = append(conditions, "pr.status = "+addArgument(string(*input.Status)))
	}
	if input.Search != "" {
		placeholder := addArgument("%" + input.Search + "%")
		conditions = append(conditions, "(pr.request_code ILIKE "+placeholder+
			" OR pr.title ILIKE "+placeholder+" OR u.display_name ILIKE "+placeholder+")")
	}
	if input.Department != "" {
		conditions = append(conditions, "d.name ILIKE "+addArgument("%"+input.Department+"%"))
	}
	if input.CostCenter != "" {
		conditions = append(conditions, "pr.cost_center ILIKE "+addArgument("%"+input.CostCenter+"%"))
	}
	if input.Requester != "" {
		conditions = append(conditions, "u.display_name ILIKE "+addArgument("%"+input.Requester+"%"))
	}
	if input.From != "" {
		conditions = append(conditions, "pr.created_at >= "+addArgument(input.From)+"::date")
	}
	if input.To != "" {
		conditions = append(conditions, "pr.created_at < ("+addArgument(input.To)+"::date + interval '1 day')")
	}
	if input.MinAmount != "" {
		conditions = append(conditions, "pr.total_amount >= "+addArgument(input.MinAmount)+"::numeric")
	}
	if input.MaxAmount != "" {
		conditions = append(conditions, "pr.total_amount <= "+addArgument(input.MaxAmount)+"::numeric")
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	limitPlaceholder := addArgument(input.PageSize)
	offsetPlaceholder := addArgument((input.Page - 1) * input.PageSize)
	orderColumn := map[string]string{
		"createdAt": "pr.created_at",
		"updatedAt": "pr.updated_at",
		"amount":    "pr.total_amount",
		"code":      "pr.request_code",
	}[input.Sort]
	if orderColumn == "" {
		orderColumn = "pr.created_at"
	}
	direction := "DESC"
	if input.Direction == "asc" {
		direction = "ASC"
	}

	rows, err := s.database.Query(ctx, `
		SELECT
			pr.id,
			pr.request_code,
			pr.requester_id,
			u.display_name,
			pr.department_id,
			d.name,
			pr.title,
			pr.reason,
			pr.currency,
			pr.total_amount::text,
			pr.cost_center,
			pr.status,
			pr.version,
			pr.created_at,
			pr.updated_at,
			count(*) OVER()
		FROM purchase_requests pr
		JOIN users u ON u.id = pr.requester_id
		JOIN departments d ON d.id = pr.department_id
		`+where+`
		ORDER BY `+orderColumn+` `+direction+`, pr.id `+direction+`
		LIMIT `+limitPlaceholder+` OFFSET `+offsetPlaceholder,
		args...,
	)
	if err != nil {
		return ListResult{}, fmt.Errorf("list purchase requests: %w", err)
	}
	defer rows.Close()

	result := ListResult{
		Items:    make([]PurchaseRequest, 0),
		Page:     input.Page,
		PageSize: input.PageSize,
	}
	for rows.Next() {
		var request PurchaseRequest
		if err = rows.Scan(
			&request.ID,
			&request.RequestCode,
			&request.RequesterID,
			&request.RequesterName,
			&request.DepartmentID,
			&request.DepartmentName,
			&request.Title,
			&request.Reason,
			&request.Currency,
			&request.TotalAmount,
			&request.CostCenter,
			&request.Status,
			&request.Version,
			&request.CreatedAt,
			&request.UpdatedAt,
			&result.Total,
		); err != nil {
			return ListResult{}, fmt.Errorf("scan purchase request list: %w", err)
		}
		result.Items = append(result.Items, request)
	}
	if err = rows.Err(); err != nil {
		return ListResult{}, fmt.Errorf("iterate purchase request list: %w", err)
	}
	if result.Total > 0 {
		result.Pages = int((result.Total + int64(input.PageSize) - 1) / int64(input.PageSize))
	}
	return result, nil
}

func (s *Store) Get(
	ctx context.Context,
	principal auth.Principal,
	requestID string,
) (PurchaseRequest, error) {
	scope, err := ScopeFor(principal)
	if err != nil {
		return PurchaseRequest{}, err
	}
	user, err := s.ensureUser(ctx, principal)
	if err != nil {
		return PurchaseRequest{}, err
	}

	request, err := s.getUnscoped(ctx, requestID)
	if err != nil {
		return PurchaseRequest{}, err
	}
	switch scope {
	case ScopeOwn:
		if request.RequesterID != user.ID {
			return PurchaseRequest{}, ErrNotFound
		}
	case ScopeDepartment:
		if request.DepartmentID != user.DepartmentID {
			return PurchaseRequest{}, ErrNotFound
		}
	case ScopeFinance:
		if request.Status != StatusManagerApproved &&
			request.Status != StatusApproved &&
			request.Status != StatusRejected {
			return PurchaseRequest{}, ErrNotFound
		}
		var organizationID string
		err = s.database.QueryRow(ctx,
			"SELECT organization_id FROM departments WHERE id = $1",
			request.DepartmentID,
		).Scan(&organizationID)
		if err != nil || organizationID != user.OrganizationID {
			return PurchaseRequest{}, ErrNotFound
		}
	case ScopeAll:
	}
	return request, nil
}

func (s *Store) Timeline(
	ctx context.Context,
	principal auth.Principal,
	requestID string,
	input TimelineInput,
) (TimelineResult, error) {
	if _, err := s.Get(ctx, principal, requestID); err != nil {
		return TimelineResult{}, err
	}

	rows, err := s.database.Query(ctx, `
		SELECT
			pe.id,
			pe.event_type,
			pe.from_status,
			pe.to_status,
			u.display_name,
			pe.actor_roles,
			pe.comment,
			pe.occurred_at,
			pe.correlation_id,
			count(*) OVER()
		FROM process_events pe
		JOIN users u ON u.id = pe.actor_id
		WHERE pe.purchase_request_id = $1
		ORDER BY pe.occurred_at DESC, pe.id DESC
		LIMIT $2 OFFSET $3
	`, requestID, input.PageSize, (input.Page-1)*input.PageSize)
	if err != nil {
		return TimelineResult{}, fmt.Errorf("list purchase request timeline: %w", err)
	}
	defer rows.Close()

	result := TimelineResult{
		Items:    make([]TimelineEvent, 0),
		Page:     input.Page,
		PageSize: input.PageSize,
	}
	for rows.Next() {
		var (
			event         TimelineEvent
			fromStatus    *string
			comment       *string
			correlationID *string
		)
		if err = rows.Scan(
			&event.ID,
			&event.EventType,
			&fromStatus,
			&event.ToStatus,
			&event.ActorName,
			&event.ActorRoles,
			&comment,
			&event.OccurredAt,
			&correlationID,
			&result.Total,
		); err != nil {
			return TimelineResult{}, fmt.Errorf("scan purchase request timeline: %w", err)
		}
		if fromStatus != nil {
			status := Status(*fromStatus)
			event.FromStatus = &status
		}
		event.Comment = comment
		event.CorrelationID = correlationID
		result.Items = append(result.Items, event)
	}
	if err = rows.Err(); err != nil {
		return TimelineResult{}, fmt.Errorf("iterate purchase request timeline: %w", err)
	}
	if result.Total > 0 {
		result.Pages = int((result.Total + int64(input.PageSize) - 1) / int64(input.PageSize))
	}
	return result, nil
}

func (s *Store) ListComments(
	ctx context.Context,
	principal auth.Principal,
	requestID string,
) (CommentList, error) {
	if _, err := s.Get(ctx, principal, requestID); err != nil {
		return CommentList{}, err
	}

	rows, err := s.database.Query(ctx, `
		SELECT
			pe.id,
			pe.comment,
			pe.actor_id,
			u.display_name,
			pe.actor_roles,
			pe.occurred_at,
			count(*) OVER()
		FROM process_events pe
		JOIN users u ON u.id = pe.actor_id
		WHERE pe.purchase_request_id = $1
		  AND pe.event_type = 'COMMENT_ADDED'
		ORDER BY pe.occurred_at ASC, pe.id ASC
	`, requestID)
	if err != nil {
		return CommentList{}, fmt.Errorf("list purchase request comments: %w", err)
	}
	defer rows.Close()

	result := CommentList{Items: make([]Comment, 0)}
	for rows.Next() {
		var comment Comment
		if err = rows.Scan(
			&comment.ID,
			&comment.Body,
			&comment.AuthorID,
			&comment.AuthorName,
			&comment.AuthorRoles,
			&comment.CreatedAt,
			&result.Total,
		); err != nil {
			return CommentList{}, fmt.Errorf("scan purchase request comment: %w", err)
		}
		result.Items = append(result.Items, comment)
	}
	if err = rows.Err(); err != nil {
		return CommentList{}, fmt.Errorf("iterate purchase request comments: %w", err)
	}
	return result, nil
}

func (s *Store) AddComment(
	ctx context.Context,
	principal auth.Principal,
	requestID string,
	input CommentInput,
) (Comment, error) {
	if hasRole(principal.Roles, "auditor") ||
		(!hasRole(principal.Roles, "employee") &&
			!hasRole(principal.Roles, "department_manager") &&
			!hasRole(principal.Roles, "finance")) {
		return Comment{}, ErrForbidden
	}
	if err := ValidateComment(&input); err != nil {
		return Comment{}, err
	}

	tx, err := s.database.Begin(ctx)
	if err != nil {
		return Comment{}, fmt.Errorf("begin purchase request comment: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return Comment{}, err
	}
	current, err := lockRequest(ctx, tx, requestID)
	if err != nil {
		return Comment{}, err
	}
	scope, err := ScopeFor(principal)
	if err != nil {
		return Comment{}, err
	}
	if !canAccessLockedRequest(scope, user, current) {
		return Comment{}, ErrNotFound
	}

	comment := Comment{
		Body:        input.Body,
		AuthorID:    user.ID,
		AuthorName:  principal.Username,
		AuthorRoles: principal.Roles,
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO process_events (
			purchase_request_id,
			event_type,
			from_status,
			to_status,
			actor_id,
			actor_roles,
			comment,
			correlation_id
		)
		VALUES ($1, 'COMMENT_ADDED', $2, $2, $3, $4, $5, NULLIF($6, ''))
		RETURNING id, occurred_at
	`, requestID, current.Status, user.ID, principal.Roles, input.Body, input.CorrelationID).Scan(
		&comment.ID,
		&comment.CreatedAt,
	)
	if err != nil {
		return Comment{}, fmt.Errorf("insert purchase request comment: %w", err)
	}
	if err = insertAudit(
		ctx, tx, requestID, "COMMENT_ADDED", user.ID, principal.Roles,
		string(current.Status), current.Status, input.CorrelationID,
	); err != nil {
		return Comment{}, err
	}
	if err = queueCommentNotification(ctx, tx, requestID, current, user.ID); err != nil {
		return Comment{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Comment{}, fmt.Errorf("commit purchase request comment: %w", err)
	}
	return comment, nil
}

func (s *Store) TaskSummary(
	ctx context.Context,
	principal auth.Principal,
) (WorkSummary, error) {
	user, err := s.ensureUser(ctx, principal)
	if err != nil {
		return WorkSummary{}, err
	}

	canOwn := CanCreate(principal)
	isManager := hasRole(principal.Roles, "department_manager")
	isFinance := hasRole(principal.Roles, "finance")
	isAuditor := hasRole(principal.Roles, "auditor")
	rows, err := s.database.Query(ctx, `
		WITH scoped_tasks AS (
			SELECT
				pr.id,
				pr.request_code,
				pr.title,
				u.display_name AS requester_name,
				d.name AS department_name,
				pr.status,
				CASE
					WHEN $2::boolean AND pr.requester_id = $1
						AND pr.status IN ('DRAFT', 'CHANGES_REQUESTED') THEN 'COMPLETE_REQUEST'
					WHEN $3::boolean AND pr.department_id = $5
						AND pr.requester_id <> $1 AND pr.status = 'SUBMITTED' THEN 'MANAGER_REVIEW'
					WHEN $4::boolean AND d.organization_id = $6
						AND pr.requester_id <> $1 AND pr.status = 'MANAGER_APPROVED' THEN 'FINANCE_REVIEW'
					WHEN $7::boolean THEN 'SLA_MONITOR'
				END AS task_type,
				pr.currency,
				pr.total_amount::text,
				CASE
					WHEN pr.submitted_at IS NULL THEN NULL
					ELSE COALESCE(
						pr.sla_due_at,
						pr.submitted_at + make_interval(hours => COALESCE(sp.target_hours, 72))
					)
				END AS due_at,
				pr.updated_at
			FROM purchase_requests pr
			JOIN users u ON u.id = pr.requester_id
			JOIN departments d ON d.id = pr.department_id
			LEFT JOIN reporting.sla_policies sp
				ON sp.organization_id = d.organization_id
			   AND sp.process_name = 'PURCHASE_REQUEST_APPROVAL'
			   AND sp.active
			WHERE
				($2::boolean AND pr.requester_id = $1 AND pr.status IN ('DRAFT', 'CHANGES_REQUESTED'))
				OR ($3::boolean AND pr.department_id = $5 AND pr.requester_id <> $1 AND pr.status = 'SUBMITTED')
				OR ($4::boolean AND d.organization_id = $6 AND pr.requester_id <> $1 AND pr.status = 'MANAGER_APPROVED')
				OR ($7::boolean AND pr.status IN ('SUBMITTED', 'MANAGER_APPROVED', 'CHANGES_REQUESTED'))
		)
		SELECT
			id,
			request_code,
			title,
			requester_name,
			department_name,
			status,
			task_type,
			currency,
			total_amount,
			due_at,
			COALESCE(due_at < now(), false) AS overdue,
			CASE
				WHEN due_at < now() THEN 'OVERDUE'
				WHEN due_at <= now() + interval '24 hours' THEN 'DUE_SOON'
				ELSE 'NORMAL'
			END AS urgency,
			updated_at
		FROM scoped_tasks
		WHERE task_type IS NOT NULL
		ORDER BY overdue DESC, due_at ASC NULLS LAST, updated_at DESC, id DESC
		LIMIT 100
	`, user.ID, canOwn, isManager, isFinance, user.DepartmentID, user.OrganizationID, isAuditor)
	if err != nil {
		return WorkSummary{}, fmt.Errorf("list work center tasks: %w", err)
	}
	defer rows.Close()

	result := WorkSummary{Items: make([]WorkTask, 0)}
	for rows.Next() {
		var task WorkTask
		if err = rows.Scan(
			&task.PurchaseRequestID,
			&task.RequestCode,
			&task.Title,
			&task.RequesterName,
			&task.DepartmentName,
			&task.Status,
			&task.TaskType,
			&task.Currency,
			&task.TotalAmount,
			&task.DueAt,
			&task.Overdue,
			&task.Urgency,
			&task.UpdatedAt,
		); err != nil {
			return WorkSummary{}, fmt.Errorf("scan work center task: %w", err)
		}
		result.Items = append(result.Items, task)
		if task.Overdue {
			result.OverdueCount++
		} else if task.Urgency == "DUE_SOON" {
			result.DueSoonCount++
		}
	}
	if err = rows.Err(); err != nil {
		return WorkSummary{}, fmt.Errorf("iterate work center tasks: %w", err)
	}
	result.Total = len(result.Items)
	return result, nil
}

func (s *Store) ListSuppliers(
	ctx context.Context,
	principal auth.Principal,
) (SupplierList, error) {
	if !hasRole(principal.Roles, "finance") && !hasRole(principal.Roles, "auditor") {
		return SupplierList{}, ErrForbidden
	}
	user, err := s.ensureUser(ctx, principal)
	if err != nil {
		return SupplierList{}, err
	}
	rows, err := s.database.Query(ctx, `
		SELECT id, code, name, COALESCE(tax_code, ''), COALESCE(contact_name, ''),
			COALESCE(email, ''), COALESCE(phone, ''), COALESCE(address, ''),
			COALESCE(bank_name, ''), COALESCE(bank_account_number, ''),
			COALESCE(contract_reference, ''), COALESCE(contract_expires_on::text, ''),
			compliance_status, COALESCE(performance_score::text, ''), COALESCE(business_note, ''),
			status, risk_level, version,
			created_at, updated_at
		FROM suppliers
		WHERE organization_id = $1
		ORDER BY CASE status WHEN 'ACTIVE' THEN 1 ELSE 2 END,
			CASE risk_level WHEN 'HIGH' THEN 1 WHEN 'MEDIUM' THEN 2 ELSE 3 END,
			name, id
	`, user.OrganizationID)
	if err != nil {
		return SupplierList{}, fmt.Errorf("list suppliers: %w", err)
	}
	defer rows.Close()
	result := SupplierList{Items: make([]Supplier, 0), CanManage: hasRole(principal.Roles, "finance")}
	for rows.Next() {
		var supplier Supplier
		if err = rows.Scan(
			&supplier.ID, &supplier.Code, &supplier.Name, &supplier.TaxCode,
			&supplier.ContactName, &supplier.Email, &supplier.Phone,
			&supplier.Address, &supplier.BankName, &supplier.BankAccountNumber,
			&supplier.ContractReference, &supplier.ContractExpiresOn, &supplier.ComplianceStatus,
			&supplier.PerformanceScore, &supplier.BusinessNote,
			&supplier.Status, &supplier.RiskLevel, &supplier.Version, &supplier.CreatedAt, &supplier.UpdatedAt,
		); err != nil {
			return SupplierList{}, fmt.Errorf("scan supplier: %w", err)
		}
		result.Items = append(result.Items, supplier)
	}
	if err = rows.Err(); err != nil {
		return SupplierList{}, fmt.Errorf("iterate suppliers: %w", err)
	}
	result.Total = len(result.Items)
	return result, nil
}

func (s *Store) CreateSupplier(
	ctx context.Context,
	principal auth.Principal,
	input SupplierInput,
	correlationID string,
) (Supplier, error) {
	if !hasRole(principal.Roles, "finance") {
		return Supplier{}, ErrForbidden
	}
	if err := ValidateSupplierInput(&input); err != nil {
		return Supplier{}, err
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return Supplier{}, fmt.Errorf("begin create supplier: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return Supplier{}, err
	}
	var supplierID string
	err = tx.QueryRow(ctx, `
		INSERT INTO suppliers (
			organization_id, code, name, tax_code, contact_name, email, phone,
			address, bank_name, bank_account_number, contract_reference,
			contract_expires_on, compliance_status, performance_score, business_note,
			status, risk_level, created_by
		)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''),
			NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''),
			NULLIF($11, ''), NULLIF($12, '')::date, $13, NULLIF($14, '')::numeric,
			NULLIF($15, ''), $16, $17, $18)
		RETURNING id
	`, user.OrganizationID, input.Code, input.Name, input.TaxCode, input.ContactName,
		input.Email, input.Phone, input.Address, input.BankName, input.BankAccountNumber,
		input.ContractReference, input.ContractExpiresOn, input.ComplianceStatus,
		input.PerformanceScore, input.BusinessNote, input.Status, input.RiskLevel, user.ID).Scan(&supplierID)
	if err != nil {
		if isUniqueViolation(err) {
			return Supplier{}, ErrSupplierConflict
		}
		return Supplier{}, fmt.Errorf("insert supplier: %w", err)
	}
	if err = insertResourceAudit(
		ctx, tx, "supplier", supplierID, "SUPPLIER_CREATED", user.ID,
		principal.Roles, "", input.Status, correlationID,
	); err != nil {
		return Supplier{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Supplier{}, fmt.Errorf("commit create supplier: %w", err)
	}
	return s.getSupplier(ctx, user.OrganizationID, supplierID)
}

func (s *Store) UpdateSupplier(
	ctx context.Context,
	principal auth.Principal,
	supplierID string,
	input UpdateSupplierInput,
	correlationID string,
) (Supplier, error) {
	if !hasRole(principal.Roles, "finance") {
		return Supplier{}, ErrForbidden
	}
	if err := ValidateUpdateSupplierInput(&input); err != nil {
		return Supplier{}, err
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return Supplier{}, fmt.Errorf("begin update supplier: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return Supplier{}, err
	}
	var currentVersion int64
	var currentStatus string
	err = tx.QueryRow(ctx, `
		SELECT version, status FROM suppliers
		WHERE id = $1 AND organization_id = $2
		FOR UPDATE
	`, supplierID, user.OrganizationID).Scan(&currentVersion, &currentStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return Supplier{}, ErrSupplierNotFound
	}
	if err != nil {
		return Supplier{}, fmt.Errorf("lock supplier: %w", err)
	}
	if currentVersion != input.ExpectedVersion {
		return Supplier{}, ErrSupplierVersion
	}
	_, err = tx.Exec(ctx, `
		UPDATE suppliers
		SET code = $2, name = $3, tax_code = NULLIF($4, ''),
			contact_name = NULLIF($5, ''), email = NULLIF($6, ''), phone = NULLIF($7, ''),
			address = NULLIF($8, ''), bank_name = NULLIF($9, ''), bank_account_number = NULLIF($10, ''),
			contract_reference = NULLIF($11, ''), contract_expires_on = NULLIF($12, '')::date,
			compliance_status = $13, performance_score = NULLIF($14, '')::numeric,
			business_note = NULLIF($15, ''), status = $16, risk_level = $17,
			version = version + 1, updated_at = now()
		WHERE id = $1
	`, supplierID, input.Code, input.Name, input.TaxCode, input.ContactName,
		input.Email, input.Phone, input.Address, input.BankName, input.BankAccountNumber,
		input.ContractReference, input.ContractExpiresOn, input.ComplianceStatus,
		input.PerformanceScore, input.BusinessNote, input.Status, input.RiskLevel)
	if err != nil {
		if isUniqueViolation(err) {
			return Supplier{}, ErrSupplierConflict
		}
		return Supplier{}, fmt.Errorf("update supplier: %w", err)
	}
	if err = insertResourceAudit(
		ctx, tx, "supplier", supplierID, "SUPPLIER_UPDATED", user.ID,
		principal.Roles, currentStatus, input.Status, correlationID,
	); err != nil {
		return Supplier{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Supplier{}, fmt.Errorf("commit update supplier: %w", err)
	}
	return s.getSupplier(ctx, user.OrganizationID, supplierID)
}

func (s *Store) getSupplier(ctx context.Context, organizationID, supplierID string) (Supplier, error) {
	var supplier Supplier
	err := s.database.QueryRow(ctx, `
		SELECT id, code, name, COALESCE(tax_code, ''), COALESCE(contact_name, ''),
			COALESCE(email, ''), COALESCE(phone, ''), COALESCE(address, ''),
			COALESCE(bank_name, ''), COALESCE(bank_account_number, ''),
			COALESCE(contract_reference, ''), COALESCE(contract_expires_on::text, ''),
			compliance_status, COALESCE(performance_score::text, ''), COALESCE(business_note, ''),
			status, risk_level, version,
			created_at, updated_at
		FROM suppliers WHERE id = $1 AND organization_id = $2
	`, supplierID, organizationID).Scan(
		&supplier.ID, &supplier.Code, &supplier.Name, &supplier.TaxCode,
		&supplier.ContactName, &supplier.Email, &supplier.Phone,
		&supplier.Address, &supplier.BankName, &supplier.BankAccountNumber,
		&supplier.ContractReference, &supplier.ContractExpiresOn, &supplier.ComplianceStatus,
		&supplier.PerformanceScore, &supplier.BusinessNote,
		&supplier.Status, &supplier.RiskLevel, &supplier.Version, &supplier.CreatedAt, &supplier.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Supplier{}, ErrSupplierNotFound
	}
	if err != nil {
		return Supplier{}, fmt.Errorf("get supplier: %w", err)
	}
	return supplier, nil
}

func (s *Store) OperationsBoard(
	ctx context.Context,
	principal auth.Principal,
) (OperationsBoard, error) {
	isEmployee := hasRole(principal.Roles, "employee")
	isManager := hasRole(principal.Roles, "department_manager")
	isFinance := hasRole(principal.Roles, "finance")
	isAuditor := hasRole(principal.Roles, "auditor")
	if !isEmployee && !isManager && !isFinance && !isAuditor {
		return OperationsBoard{}, ErrForbidden
	}
	user, err := s.ensureUser(ctx, principal)
	if err != nil {
		return OperationsBoard{}, err
	}
	rows, err := s.database.Query(ctx, `
		SELECT
			COALESCE(po.id::text, ''),
			pr.id,
			pr.request_code,
			pr.title,
			ru.display_name,
			d.name,
			pr.currency,
			pr.total_amount::text,
			po.order_code,
			po.supplier_id,
			s.code,
			s.name,
			po.external_reference,
			po.expected_delivery_on::text,
			po.actual_delivery_on::text,
			COALESCE(po.status, 'AWAITING_ORDER'),
			po.note,
			COALESCE(po.version, 0),
			po.ordered_at,
			po.received_at,
			po.cancelled_at,
			po.cancellation_reason,
			COALESCE((SELECT count(*) FROM purchase_order_receipts por WHERE por.purchase_order_id = po.id), 0),
			COALESCE(po.status IN ('ORDERED', 'PARTIALLY_RECEIVED', 'RECEIPT_EXCEPTION') AND po.expected_delivery_on < CURRENT_DATE, false),
			($4::boolean AND po.id IS NULL AND d.organization_id = $6),
			COALESCE(
				po.status IN ('ORDERED', 'PARTIALLY_RECEIVED', 'RECEIPT_EXCEPTION') AND (
					(($2::boolean OR $3::boolean) AND pr.requester_id = $1)
					OR ($3::boolean AND pr.department_id = $5)
				),
				false
			),
			COALESCE($4::boolean AND po.status IN ('ORDERED', 'PARTIALLY_RECEIVED', 'RECEIPT_EXCEPTION'), false)
		FROM purchase_requests pr
		JOIN users ru ON ru.id = pr.requester_id
		JOIN departments d ON d.id = pr.department_id
		LEFT JOIN purchase_orders po ON po.purchase_request_id = pr.id
		LEFT JOIN suppliers s ON s.id = po.supplier_id
		WHERE pr.status = 'APPROVED'
		  AND (
			$7::boolean
			OR ($4::boolean AND d.organization_id = $6)
			OR ($3::boolean AND pr.department_id = $5)
			OR ($2::boolean AND pr.requester_id = $1)
		  )
		ORDER BY
			CASE WHEN po.status IN ('ORDERED', 'PARTIALLY_RECEIVED', 'RECEIPT_EXCEPTION') AND po.expected_delivery_on < CURRENT_DATE THEN 1 ELSE 2 END,
			CASE COALESCE(po.status, 'AWAITING_ORDER')
				WHEN 'RECEIPT_EXCEPTION' THEN 1 WHEN 'AWAITING_ORDER' THEN 2
				WHEN 'ORDERED' THEN 3 WHEN 'PARTIALLY_RECEIVED' THEN 4
				WHEN 'RECEIVED' THEN 5 ELSE 6 END,
			COALESCE(po.expected_delivery_on, CURRENT_DATE + 3650), pr.updated_at DESC
		LIMIT 200
	`, user.ID, isEmployee, isManager, isFinance, user.DepartmentID, user.OrganizationID, isAuditor)
	if err != nil {
		return OperationsBoard{}, fmt.Errorf("list procurement operations: %w", err)
	}
	defer rows.Close()
	result := OperationsBoard{Items: make([]PurchaseOrder, 0)}
	for rows.Next() {
		var order PurchaseOrder
		if err = rows.Scan(
			&order.ID, &order.PurchaseRequestID, &order.RequestCode, &order.RequestTitle,
			&order.RequesterName, &order.DepartmentName, &order.Currency, &order.TotalAmount,
			&order.OrderCode, &order.SupplierID, &order.SupplierCode, &order.SupplierName,
			&order.ExternalReference, &order.ExpectedDeliveryOn, &order.ActualDeliveryOn,
			&order.Status, &order.Note, &order.Version, &order.OrderedAt, &order.ReceivedAt,
			&order.CancelledAt, &order.CancellationReason, &order.ReceiptCount,
			&order.DeliveryOverdue, &order.CanPlaceOrder, &order.CanConfirmReceipt, &order.CanManageOrder,
		); err != nil {
			return OperationsBoard{}, fmt.Errorf("scan procurement operation: %w", err)
		}
		result.Items = append(result.Items, order)
		switch order.Status {
		case "AWAITING_ORDER":
			result.AwaitingOrderCount++
		case "ORDERED":
			result.InDeliveryCount++
			if order.DeliveryOverdue {
				result.OverdueDeliveryCount++
			}
		case "RECEIVED":
			result.ReceivedCount++
		case "PARTIALLY_RECEIVED":
			result.PartialCount++
		case "RECEIPT_EXCEPTION":
			result.ExceptionCount++
		case "CANCELLED":
			result.CancelledCount++
		}
	}
	if err = rows.Err(); err != nil {
		return OperationsBoard{}, fmt.Errorf("iterate procurement operations: %w", err)
	}
	result.Total = len(result.Items)
	return result, nil
}

func (s *Store) CreatePurchaseOrder(
	ctx context.Context,
	principal auth.Principal,
	input CreatePurchaseOrderInput,
) (PurchaseOrder, error) {
	if !hasRole(principal.Roles, "finance") || hasRole(principal.Roles, "auditor") {
		return PurchaseOrder{}, ErrForbidden
	}
	if err := ValidateCreatePurchaseOrder(&input); err != nil {
		return PurchaseOrder{}, err
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("begin create purchase order: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return PurchaseOrder{}, err
	}
	current, err := lockRequest(ctx, tx, input.PurchaseRequestID)
	if err != nil {
		return PurchaseOrder{}, err
	}
	if current.OrganizationID != user.OrganizationID || current.Status != StatusApproved {
		return PurchaseOrder{}, ErrInvalidFulfillment
	}
	var existingRequestID string
	err = tx.QueryRow(ctx, `
		SELECT purchase_request_id FROM purchase_orders WHERE idempotency_key = $1
	`, input.IdempotencyKey).Scan(&existingRequestID)
	switch {
	case err == nil:
		if existingRequestID != input.PurchaseRequestID {
			return PurchaseOrder{}, ErrPurchaseOrderConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return PurchaseOrder{}, fmt.Errorf("commit purchase order replay: %w", err)
		}
		return s.loadPurchaseOrder(ctx, input.PurchaseRequestID)
	case !errors.Is(err, pgx.ErrNoRows):
		return PurchaseOrder{}, fmt.Errorf("check purchase order idempotency: %w", err)
	}
	var supplierActive bool
	err = tx.QueryRow(ctx, `
		SELECT status = 'ACTIVE' FROM suppliers
		WHERE id = $1 AND organization_id = $2
	`, input.SupplierID, user.OrganizationID).Scan(&supplierActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return PurchaseOrder{}, ErrSupplierNotFound
	}
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("get purchase order supplier: %w", err)
	}
	if !supplierActive {
		return PurchaseOrder{}, ErrInvalidFulfillment
	}
	var orderID string
	err = tx.QueryRow(ctx, `
		INSERT INTO purchase_orders (
			organization_id, purchase_request_id, supplier_id, order_code,
			external_reference, expected_delivery_on, note, ordered_by, idempotency_key
		)
		VALUES (
			$1, $2, $3,
			'PO-' || to_char(CURRENT_DATE, 'YYYY') || '-' || lpad(nextval('purchase_order_code_seq')::text, 6, '0'),
			NULLIF($4, ''), $5::date, NULLIF($6, ''), $7, $8
		)
		RETURNING id
	`, user.OrganizationID, input.PurchaseRequestID, input.SupplierID,
		input.ExternalReference, input.ExpectedDeliveryOn, input.Note, user.ID,
		input.IdempotencyKey).Scan(&orderID)
	if err != nil {
		if isUniqueViolation(err) {
			return PurchaseOrder{}, ErrPurchaseOrderConflict
		}
		return PurchaseOrder{}, fmt.Errorf("insert purchase order: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO process_events (
			purchase_request_id, event_type, from_status, to_status, actor_id,
			actor_roles, comment, correlation_id
		)
		VALUES ($1, 'ORDER_PLACED', 'APPROVED', 'APPROVED', $2, $3, NULLIF($4, ''), NULLIF($5, ''))
	`, input.PurchaseRequestID, user.ID, principal.Roles, input.Note, input.CorrelationID)
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("insert order-placed event: %w", err)
	}
	if err = insertAudit(ctx, tx, input.PurchaseRequestID, "ORDER_PLACED", user.ID,
		principal.Roles, string(StatusApproved), StatusApproved, input.CorrelationID); err != nil {
		return PurchaseOrder{}, err
	}
	if err = insertResourceAudit(ctx, tx, "purchase_order", orderID, "PURCHASE_ORDER_CREATED",
		user.ID, principal.Roles, "", "ORDERED", input.CorrelationID); err != nil {
		return PurchaseOrder{}, err
	}
	if err = notifications.Queue(ctx, tx, notifications.QueueInput{
		EventType: "ORDER_PLACED", ResourceType: "purchase_request",
		ResourceID: input.PurchaseRequestID, OrganizationID: current.OrganizationID,
		DepartmentID: current.DepartmentID, RecipientUserID: current.RequesterID,
		ActorID: user.ID, Title: "Đơn mua hàng đã được tạo",
		Body: current.RequestCode + " - " + current.Title,
	}); err != nil {
		return PurchaseOrder{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return PurchaseOrder{}, fmt.Errorf("commit purchase order: %w", err)
	}
	return s.loadPurchaseOrder(ctx, input.PurchaseRequestID)
}

func (s *Store) ConfirmReceipt(
	ctx context.Context,
	principal auth.Principal,
	requestID string,
	input ConfirmReceiptInput,
) (PurchaseOrder, error) {
	if err := ValidateConfirmReceipt(&input); err != nil {
		return PurchaseOrder{}, err
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("begin confirm receipt: %w", err)
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
	`, requestID, user.OrganizationID).Scan(
		&orderID, &status, &version, &requesterID, &departmentID, &requestCode, &requestTitle,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PurchaseOrder{}, ErrPurchaseOrderNotFound
	}
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("lock purchase order: %w", err)
	}
	canReceive := (requesterID == user.ID &&
		(hasRole(principal.Roles, "employee") || hasRole(principal.Roles, "department_manager"))) ||
		(hasRole(principal.Roles, "department_manager") && departmentID == user.DepartmentID)
	if !canReceive {
		return PurchaseOrder{}, ErrForbidden
	}
	if status != "ORDERED" {
		return PurchaseOrder{}, ErrInvalidFulfillment
	}
	if version != input.ExpectedVersion {
		return PurchaseOrder{}, ErrVersionConflict
	}
	_, err = tx.Exec(ctx, `
		UPDATE purchase_orders
		SET status = 'RECEIVED', actual_delivery_on = $2::date, received_by = $3,
			received_at = now(), version = version + 1, updated_at = now()
		WHERE id = $1
	`, orderID, input.ActualDeliveryOn, user.ID)
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("confirm purchase order receipt: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO process_events (
			purchase_request_id, event_type, from_status, to_status, actor_id,
			actor_roles, correlation_id
		)
		VALUES ($1, 'DELIVERY_RECEIVED', 'APPROVED', 'APPROVED', $2, $3, NULLIF($4, ''))
	`, requestID, user.ID, principal.Roles, input.CorrelationID)
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("insert delivery-received event: %w", err)
	}
	if err = insertAudit(ctx, tx, requestID, "DELIVERY_RECEIVED", user.ID,
		principal.Roles, string(StatusApproved), StatusApproved, input.CorrelationID); err != nil {
		return PurchaseOrder{}, err
	}
	if err = insertResourceAudit(ctx, tx, "purchase_order", orderID, "DELIVERY_RECEIVED",
		user.ID, principal.Roles, "ORDERED", "RECEIVED", input.CorrelationID); err != nil {
		return PurchaseOrder{}, err
	}
	if err = notifications.Queue(ctx, tx, notifications.QueueInput{
		EventType: "DELIVERY_RECEIVED", ResourceType: "purchase_request",
		ResourceID: requestID, OrganizationID: user.OrganizationID,
		DepartmentID: departmentID, RecipientRole: "finance", ActorID: user.ID,
		Title: "Đơn hàng đã được xác nhận nhận hàng", Body: requestCode + " - " + requestTitle,
	}); err != nil {
		return PurchaseOrder{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return PurchaseOrder{}, fmt.Errorf("commit receipt confirmation: %w", err)
	}
	return s.loadPurchaseOrder(ctx, requestID)
}

func (s *Store) loadPurchaseOrder(ctx context.Context, requestID string) (PurchaseOrder, error) {
	var order PurchaseOrder
	err := s.database.QueryRow(ctx, `
		SELECT
			po.id, pr.id, pr.request_code, pr.title, ru.display_name, d.name,
			pr.currency, pr.total_amount::text, po.order_code, po.supplier_id,
			s.code, s.name, po.external_reference, po.expected_delivery_on::text,
			po.actual_delivery_on::text, po.status, po.note, po.version,
			po.ordered_at, po.received_at,
			po.cancelled_at, po.cancellation_reason,
			(SELECT count(*) FROM purchase_order_receipts por WHERE por.purchase_order_id = po.id),
			(po.status IN ('ORDERED', 'PARTIALLY_RECEIVED', 'RECEIPT_EXCEPTION') AND po.expected_delivery_on < CURRENT_DATE)
		FROM purchase_orders po
		JOIN purchase_requests pr ON pr.id = po.purchase_request_id
		JOIN users ru ON ru.id = pr.requester_id
		JOIN departments d ON d.id = pr.department_id
		JOIN suppliers s ON s.id = po.supplier_id
		WHERE po.purchase_request_id = $1
	`, requestID).Scan(
		&order.ID, &order.PurchaseRequestID, &order.RequestCode, &order.RequestTitle,
		&order.RequesterName, &order.DepartmentName, &order.Currency, &order.TotalAmount,
		&order.OrderCode, &order.SupplierID, &order.SupplierCode, &order.SupplierName,
		&order.ExternalReference, &order.ExpectedDeliveryOn, &order.ActualDeliveryOn,
		&order.Status, &order.Note, &order.Version, &order.OrderedAt, &order.ReceivedAt,
		&order.CancelledAt, &order.CancellationReason, &order.ReceiptCount, &order.DeliveryOverdue,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PurchaseOrder{}, ErrPurchaseOrderNotFound
	}
	if err != nil {
		return PurchaseOrder{}, fmt.Errorf("get purchase order: %w", err)
	}
	return order, nil
}

func (s *Store) GetBudgetSummary(
	ctx context.Context,
	principal auth.Principal,
	input BudgetSummaryInput,
) (BudgetSummary, error) {
	if err := ValidateBudgetSummaryInput(&input); err != nil {
		return BudgetSummary{}, err
	}
	user, err := s.ensureUser(ctx, principal)
	if err != nil {
		return BudgetSummary{}, err
	}

	switch {
	case hasRole(principal.Roles, "finance"), hasRole(principal.Roles, "auditor"):
		// Finance and audit can inspect all cost centers in their organization.
	case hasRole(principal.Roles, "department_manager"):
		var departmentCostCenter string
		err = s.database.QueryRow(ctx, `
			SELECT COALESCE(cost_center, '')
			FROM departments
			WHERE id = $1 AND active
		`, user.DepartmentID).Scan(&departmentCostCenter)
		if err != nil {
			return BudgetSummary{}, fmt.Errorf("get department cost center: %w", err)
		}
		if departmentCostCenter != input.CostCenter {
			return BudgetSummary{}, ErrBudgetNotFound
		}
	default:
		return BudgetSummary{}, ErrForbidden
	}

	summary, err := activeBudgetSummary(
		ctx, s.database, user.OrganizationID, input.CostCenter, input.Currency,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return BudgetSummary{}, ErrBudgetNotFound
	}
	if err != nil {
		return BudgetSummary{}, fmt.Errorf("get budget summary: %w", err)
	}
	return summary, nil
}

func (s *Store) BudgetCheck(
	ctx context.Context,
	principal auth.Principal,
	requestID string,
) (BudgetCheck, error) {
	request, err := s.Get(ctx, principal, requestID)
	if err != nil {
		return BudgetCheck{}, err
	}
	var organizationID string
	if err = s.database.QueryRow(ctx, `
		SELECT organization_id
		FROM departments
		WHERE id = $1
	`, request.DepartmentID).Scan(&organizationID); err != nil {
		return BudgetCheck{}, fmt.Errorf("get purchase request organization: %w", err)
	}

	var reservationState string
	summary, err := scanBudgetSummary(s.database.QueryRow(ctx, `
		SELECT
			bp.code,
			bp.starts_on,
			bp.ends_on,
			ba.cost_center,
			ba.currency,
			ba.allocated_amount::text,
			ba.reserved_amount::text,
			ba.committed_amount::text,
			(ba.allocated_amount - ba.reserved_amount - ba.committed_amount)::text,
			br.status
		FROM budget_reservations br
		JOIN budget_allocations ba ON ba.id = br.budget_allocation_id
		JOIN budget_periods bp ON bp.id = ba.budget_period_id
		WHERE br.purchase_request_id = $1
		  AND br.status IN ('RESERVED', 'COMMITTED')
		ORDER BY br.reserved_at DESC
		LIMIT 1
	`, requestID), &reservationState)
	if err == nil {
		return BudgetCheck{
			Configured:       true,
			Result:           reservationState,
			RequestedAmount:  request.TotalAmount,
			ReservationState: &reservationState,
			Summary:          &summary,
		}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return BudgetCheck{}, fmt.Errorf("get purchase request budget reservation: %w", err)
	}

	summary, err = activeBudgetSummary(
		ctx, s.database, organizationID, request.CostCenter, request.Currency,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return BudgetCheck{
			Configured:      false,
			Result:          "NOT_CONFIGURED",
			RequestedAmount: request.TotalAmount,
		}, nil
	}
	if err != nil {
		return BudgetCheck{}, fmt.Errorf("check purchase request budget: %w", err)
	}

	var enough bool
	if err = s.database.QueryRow(ctx, `
		SELECT $1::numeric <= $2::numeric
	`, request.TotalAmount, summary.AvailableAmount).Scan(&enough); err != nil {
		return BudgetCheck{}, fmt.Errorf("compare available budget: %w", err)
	}
	result := "INSUFFICIENT"
	if enough {
		result = "AVAILABLE"
	}
	return BudgetCheck{
		Configured:      true,
		Result:          result,
		RequestedAmount: request.TotalAmount,
		Summary:         &summary,
	}, nil
}

func (s *Store) BudgetDashboard(
	ctx context.Context,
	principal auth.Principal,
) (BudgetDashboard, error) {
	if !hasRole(principal.Roles, "finance") && !hasRole(principal.Roles, "auditor") {
		return BudgetDashboard{}, ErrForbidden
	}
	user, err := s.ensureUser(ctx, principal)
	if err != nil {
		return BudgetDashboard{}, err
	}
	result := BudgetDashboard{
		Allocations:  make([]BudgetAllocation, 0),
		Totals:       make([]BudgetCurrencyTotal, 0),
		Reservations: make([]BudgetReservation, 0),
		Adjustments:  make([]BudgetAdjustment, 0),
		CanManage:    hasRole(principal.Roles, "finance"),
	}

	rows, err := s.database.Query(ctx, budgetAllocationSelect+`
		WHERE bp.organization_id = $1
		  AND bp.status = 'ACTIVE'
		  AND CURRENT_DATE BETWEEN bp.starts_on AND bp.ends_on
		ORDER BY ba.cost_center, ba.currency
	`, user.OrganizationID)
	if err != nil {
		return BudgetDashboard{}, fmt.Errorf("list budget allocations: %w", err)
	}
	for rows.Next() {
		allocation, scanErr := scanBudgetAllocation(rows)
		if scanErr != nil {
			rows.Close()
			return BudgetDashboard{}, scanErr
		}
		if allocation.AlertLevel != "HEALTHY" {
			result.AlertCount++
		}
		result.Allocations = append(result.Allocations, allocation)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return BudgetDashboard{}, fmt.Errorf("iterate budget allocations: %w", err)
	}
	rows.Close()

	totalRows, err := s.database.Query(ctx, `
		SELECT
			ba.currency,
			sum(ba.allocated_amount)::text,
			sum(ba.reserved_amount)::text,
			sum(ba.committed_amount)::text,
			sum(ba.allocated_amount - ba.reserved_amount - ba.committed_amount)::text
		FROM budget_allocations ba
		JOIN budget_periods bp ON bp.id = ba.budget_period_id
		WHERE bp.organization_id = $1
		  AND bp.status = 'ACTIVE'
		  AND CURRENT_DATE BETWEEN bp.starts_on AND bp.ends_on
		GROUP BY ba.currency
		ORDER BY ba.currency
	`, user.OrganizationID)
	if err != nil {
		return BudgetDashboard{}, fmt.Errorf("summarize budget allocations: %w", err)
	}
	for totalRows.Next() {
		var total BudgetCurrencyTotal
		if err = totalRows.Scan(
			&total.Currency,
			&total.AllocatedAmount,
			&total.ReservedAmount,
			&total.CommittedAmount,
			&total.AvailableAmount,
		); err != nil {
			totalRows.Close()
			return BudgetDashboard{}, fmt.Errorf("scan budget totals: %w", err)
		}
		result.Totals = append(result.Totals, total)
	}
	if err = totalRows.Err(); err != nil {
		totalRows.Close()
		return BudgetDashboard{}, fmt.Errorf("iterate budget totals: %w", err)
	}
	totalRows.Close()

	reservationRows, err := s.database.Query(ctx, `
		SELECT
			br.id,
			pr.id,
			pr.request_code,
			pr.title,
			ba.cost_center,
			br.currency,
			br.amount::text,
			br.status,
			u.display_name,
			br.reserved_at,
			br.committed_at,
			br.released_at
		FROM budget_reservations br
		JOIN budget_allocations ba ON ba.id = br.budget_allocation_id
		JOIN budget_periods bp ON bp.id = ba.budget_period_id
		JOIN purchase_requests pr ON pr.id = br.purchase_request_id
		JOIN users u ON u.id = br.reserved_by
		WHERE bp.organization_id = $1
		ORDER BY br.updated_at DESC, br.id DESC
		LIMIT 50
	`, user.OrganizationID)
	if err != nil {
		return BudgetDashboard{}, fmt.Errorf("list budget reservations: %w", err)
	}
	for reservationRows.Next() {
		var reservation BudgetReservation
		if err = reservationRows.Scan(
			&reservation.ID,
			&reservation.PurchaseID,
			&reservation.RequestCode,
			&reservation.RequestTitle,
			&reservation.CostCenter,
			&reservation.Currency,
			&reservation.Amount,
			&reservation.Status,
			&reservation.ReservedBy,
			&reservation.ReservedAt,
			&reservation.CommittedAt,
			&reservation.ReleasedAt,
		); err != nil {
			reservationRows.Close()
			return BudgetDashboard{}, fmt.Errorf("scan budget reservation: %w", err)
		}
		result.Reservations = append(result.Reservations, reservation)
	}
	if err = reservationRows.Err(); err != nil {
		reservationRows.Close()
		return BudgetDashboard{}, fmt.Errorf("iterate budget reservations: %w", err)
	}
	reservationRows.Close()

	adjustmentRows, err := s.database.Query(ctx, `
		SELECT
			badj.id,
			ba.id,
			ba.cost_center,
			ba.currency,
			badj.previous_amount::text,
			badj.adjusted_amount::text,
			badj.reason,
			u.display_name,
			badj.created_at
		FROM budget_adjustments badj
		JOIN budget_allocations ba ON ba.id = badj.budget_allocation_id
		JOIN budget_periods bp ON bp.id = ba.budget_period_id
		JOIN users u ON u.id = badj.actor_id
		WHERE bp.organization_id = $1
		ORDER BY badj.created_at DESC, badj.id DESC
		LIMIT 50
	`, user.OrganizationID)
	if err != nil {
		return BudgetDashboard{}, fmt.Errorf("list budget adjustments: %w", err)
	}
	for adjustmentRows.Next() {
		var adjustment BudgetAdjustment
		if err = adjustmentRows.Scan(
			&adjustment.ID,
			&adjustment.AllocationID,
			&adjustment.CostCenter,
			&adjustment.Currency,
			&adjustment.PreviousAmount,
			&adjustment.AdjustedAmount,
			&adjustment.Reason,
			&adjustment.ActorName,
			&adjustment.CreatedAt,
		); err != nil {
			adjustmentRows.Close()
			return BudgetDashboard{}, fmt.Errorf("scan budget adjustment: %w", err)
		}
		result.Adjustments = append(result.Adjustments, adjustment)
	}
	if err = adjustmentRows.Err(); err != nil {
		adjustmentRows.Close()
		return BudgetDashboard{}, fmt.Errorf("iterate budget adjustments: %w", err)
	}
	adjustmentRows.Close()
	return result, nil
}

func (s *Store) AdjustBudget(
	ctx context.Context,
	principal auth.Principal,
	allocationID string,
	input AdjustBudgetInput,
) (BudgetAllocation, error) {
	if !hasRole(principal.Roles, "finance") {
		return BudgetAllocation{}, ErrForbidden
	}
	if err := ValidateAdjustBudgetInput(&input); err != nil {
		return BudgetAllocation{}, err
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return BudgetAllocation{}, fmt.Errorf("begin budget adjustment: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return BudgetAllocation{}, err
	}
	var existingAllocationID, existingActorID, existingAmount string
	err = tx.QueryRow(ctx, `
		SELECT budget_allocation_id, actor_id, adjusted_amount::text
		FROM budget_adjustments
		WHERE idempotency_key = $1
	`, input.IdempotencyKey).Scan(
		&existingAllocationID,
		&existingActorID,
		&existingAmount,
	)
	switch {
	case err == nil:
		if existingAllocationID != allocationID ||
			existingActorID != user.ID ||
			existingAmount != normalizedMoney(input.AllocatedAmount) {
			return BudgetAllocation{}, ErrIdempotencyConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return BudgetAllocation{}, fmt.Errorf("commit budget adjustment replay: %w", err)
		}
		return s.getBudgetAllocation(ctx, allocationID, user.OrganizationID)
	case !errors.Is(err, pgx.ErrNoRows):
		return BudgetAllocation{}, fmt.Errorf("check budget adjustment idempotency: %w", err)
	}

	var previousAmount, reservedAmount, committedAmount string
	var version int64
	err = tx.QueryRow(ctx, `
		SELECT
			ba.allocated_amount::text,
			ba.reserved_amount::text,
			ba.committed_amount::text,
			ba.version
		FROM budget_allocations ba
		JOIN budget_periods bp ON bp.id = ba.budget_period_id
		WHERE ba.id = $1
		  AND bp.organization_id = $2
		FOR UPDATE OF ba
	`, allocationID, user.OrganizationID).Scan(
		&previousAmount,
		&reservedAmount,
		&committedAmount,
		&version,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return BudgetAllocation{}, ErrBudgetNotFound
	}
	if err != nil {
		return BudgetAllocation{}, fmt.Errorf("lock budget allocation for adjustment: %w", err)
	}
	if version != input.ExpectedVersion {
		return BudgetAllocation{}, ErrBudgetVersionConflict
	}
	var coversUsage bool
	if err = tx.QueryRow(ctx, `
		SELECT $1::numeric >= ($2::numeric + $3::numeric)
	`, input.AllocatedAmount, reservedAmount, committedAmount).Scan(&coversUsage); err != nil {
		return BudgetAllocation{}, fmt.Errorf("validate adjusted budget amount: %w", err)
	}
	if !coversUsage {
		return BudgetAllocation{}, ErrBudgetBelowUsage
	}

	if _, err = tx.Exec(ctx, `
		UPDATE budget_allocations
		SET allocated_amount = $2::numeric,
			version = version + 1,
			updated_at = now()
		WHERE id = $1
	`, allocationID, input.AllocatedAmount); err != nil {
		return BudgetAllocation{}, fmt.Errorf("update budget allocation: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO budget_adjustments (
			budget_allocation_id,
			previous_amount,
			adjusted_amount,
			reason,
			actor_id,
			correlation_id,
			idempotency_key
		)
		VALUES ($1, $2::numeric, $3::numeric, $4, $5, NULLIF($6, ''), $7)
	`, allocationID, previousAmount, input.AllocatedAmount, input.Reason,
		user.ID, input.CorrelationID, input.IdempotencyKey)
	if err != nil {
		var databaseError *pgconn.PgError
		if errors.As(err, &databaseError) && databaseError.Code == "23505" {
			return BudgetAllocation{}, ErrIdempotencyConflict
		}
		return BudgetAllocation{}, fmt.Errorf("insert budget adjustment: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (
			resource_type,
			resource_id,
			action,
			actor_id,
			actor_roles,
			correlation_id,
			metadata
		)
		VALUES (
			'budget_allocation',
			$1,
			'BUDGET_ALLOCATION_ADJUSTED',
			$2,
			$3,
			NULLIF($4, ''),
			jsonb_build_object(
				'previousAmount', $5::text,
				'adjustedAmount', $6::text,
				'reason', $7::text
			)
		)
	`, allocationID, user.ID, principal.Roles, input.CorrelationID,
		previousAmount, input.AllocatedAmount, input.Reason); err != nil {
		return BudgetAllocation{}, fmt.Errorf("insert budget adjustment audit: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return BudgetAllocation{}, fmt.Errorf("commit budget adjustment: %w", err)
	}
	return s.getBudgetAllocation(ctx, allocationID, user.OrganizationID)
}

const budgetAllocationSelect = `
	SELECT
		ba.id,
		bp.code,
		bp.starts_on,
		bp.ends_on,
		ba.cost_center,
		ba.currency,
		ba.allocated_amount::text,
		ba.reserved_amount::text,
		ba.committed_amount::text,
		(ba.allocated_amount - ba.reserved_amount - ba.committed_amount)::text,
		CASE
			WHEN ba.allocated_amount = 0 THEN '100.00'
			ELSE round(
				((ba.reserved_amount + ba.committed_amount) / ba.allocated_amount) * 100,
				2
			)::text
		END,
		CASE
			WHEN ba.allocated_amount = 0
			  OR ba.reserved_amount + ba.committed_amount >= ba.allocated_amount
			  OR (ba.reserved_amount + ba.committed_amount) / NULLIF(ba.allocated_amount, 0) >= 0.90
				THEN 'CRITICAL'
			WHEN (ba.reserved_amount + ba.committed_amount) / NULLIF(ba.allocated_amount, 0) >= 0.75
				THEN 'WARNING'
			ELSE 'HEALTHY'
		END,
		ba.version
	FROM budget_allocations ba
	JOIN budget_periods bp ON bp.id = ba.budget_period_id
`

type budgetAllocationScanner interface {
	Scan(...any) error
}

func scanBudgetAllocation(row budgetAllocationScanner) (BudgetAllocation, error) {
	var allocation BudgetAllocation
	var startsOn, endsOn time.Time
	if err := row.Scan(
		&allocation.ID,
		&allocation.PeriodCode,
		&startsOn,
		&endsOn,
		&allocation.CostCenter,
		&allocation.Currency,
		&allocation.AllocatedAmount,
		&allocation.ReservedAmount,
		&allocation.CommittedAmount,
		&allocation.AvailableAmount,
		&allocation.Utilization,
		&allocation.AlertLevel,
		&allocation.Version,
	); err != nil {
		return BudgetAllocation{}, fmt.Errorf("scan budget allocation: %w", err)
	}
	allocation.PeriodStart = startsOn.Format(time.DateOnly)
	allocation.PeriodEnd = endsOn.Format(time.DateOnly)
	return allocation, nil
}

func (s *Store) getBudgetAllocation(
	ctx context.Context,
	allocationID string,
	organizationID string,
) (BudgetAllocation, error) {
	allocation, err := scanBudgetAllocation(s.database.QueryRow(ctx,
		budgetAllocationSelect+`
			WHERE ba.id = $1
			  AND bp.organization_id = $2
		`,
		allocationID,
		organizationID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return BudgetAllocation{}, ErrBudgetNotFound
	}
	return allocation, err
}

func normalizedMoney(value string) string {
	number, ok := new(big.Rat).SetString(value)
	if !ok {
		return value
	}
	return number.FloatString(4)
}

func (s *Store) Update(
	ctx context.Context,
	principal auth.Principal,
	requestID string,
	input UpdateInput,
	correlationID string,
) (PurchaseRequest, error) {
	if err := ValidateUpdate(&input); err != nil {
		return PurchaseRequest{}, err
	}

	tx, err := s.database.Begin(ctx)
	if err != nil {
		return PurchaseRequest{}, fmt.Errorf("begin update purchase request: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return PurchaseRequest{}, err
	}
	current, err := lockRequest(ctx, tx, requestID)
	if err != nil {
		return PurchaseRequest{}, err
	}
	if current.RequesterID != user.ID {
		return PurchaseRequest{}, ErrNotFound
	}
	if current.Status != StatusDraft && current.Status != StatusChangesRequested {
		return PurchaseRequest{}, ErrInvalidTransition
	}
	if current.Version != input.ExpectedVersion {
		return PurchaseRequest{}, ErrVersionConflict
	}
	_, err = tx.Exec(ctx, `
		UPDATE purchase_requests
		SET title = $2,
			reason = $3,
			currency = $4,
			cost_center = $5,
			version = version + 1,
			updated_at = now()
		WHERE id = $1
	`, requestID, input.Title, input.Reason, input.Currency, input.CostCenter)
	if err != nil {
		return PurchaseRequest{}, fmt.Errorf("update purchase request: %w", err)
	}
	if _, err = tx.Exec(ctx,
		"DELETE FROM purchase_request_items WHERE purchase_request_id = $1",
		requestID,
	); err != nil {
		return PurchaseRequest{}, fmt.Errorf("replace purchase request items: %w", err)
	}
	if err = insertItems(ctx, tx, requestID, input.Items); err != nil {
		return PurchaseRequest{}, err
	}
	if err = recalculateTotal(ctx, tx, requestID); err != nil {
		return PurchaseRequest{}, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO process_events (
			purchase_request_id,
			event_type,
			from_status,
			to_status,
			actor_id,
			actor_roles,
			correlation_id
		)
		VALUES ($1, 'DRAFT_UPDATED', $2, $2, $3, $4, NULLIF($5, ''))
	`, requestID, current.Status, user.ID, principal.Roles, correlationID)
	if err != nil {
		return PurchaseRequest{}, fmt.Errorf("insert draft-updated event: %w", err)
	}
	if err = insertAudit(
		ctx, tx, requestID, "DRAFT_UPDATED", user.ID, principal.Roles,
		string(current.Status), current.Status, correlationID,
	); err != nil {
		return PurchaseRequest{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return PurchaseRequest{}, fmt.Errorf("commit purchase request update: %w", err)
	}
	return s.getUnscoped(ctx, requestID)
}

func (s *Store) Transition(
	ctx context.Context,
	principal auth.Principal,
	requestID string,
	input TransitionInput,
) (PurchaseRequest, error) {
	if err := ValidateTransition(&input); err != nil {
		return PurchaseRequest{}, err
	}

	tx, err := s.database.Begin(ctx)
	if err != nil {
		return PurchaseRequest{}, fmt.Errorf("begin purchase request transition: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return PurchaseRequest{}, err
	}
	current, err := lockRequest(ctx, tx, requestID)
	if err != nil {
		return PurchaseRequest{}, err
	}

	var existingRequestID, existingActorID, existingAction string
	err = tx.QueryRow(ctx, `
		SELECT
			purchase_request_id::text,
			actor_id::text,
			COALESCE(metadata ->> 'action', '')
		FROM process_events
		WHERE idempotency_key = $1
	`, input.IdempotencyKey).Scan(&existingRequestID, &existingActorID, &existingAction)
	switch {
	case err == nil:
		if existingRequestID != requestID ||
			existingActorID != user.ID ||
			existingAction != string(input.Action) {
			return PurchaseRequest{}, ErrIdempotencyConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return PurchaseRequest{}, fmt.Errorf("commit idempotent transition replay: %w", err)
		}
		return s.getUnscoped(ctx, requestID)
	case !errors.Is(err, pgx.ErrNoRows):
		return PurchaseRequest{}, fmt.Errorf("check transition idempotency: %w", err)
	}

	decision, err := DecideTransition(
		ActorContext{
			UserID:         user.ID,
			DepartmentID:   user.DepartmentID,
			OrganizationID: user.OrganizationID,
			Roles:          principal.Roles,
		},
		RequestContext{
			RequesterID:    current.RequesterID,
			DepartmentID:   current.DepartmentID,
			OrganizationID: current.OrganizationID,
			Status:         current.Status,
		},
		input.Action,
	)
	if err != nil {
		return PurchaseRequest{}, err
	}
	if current.Version != input.ExpectedVersion {
		return PurchaseRequest{}, ErrVersionConflict
	}
	if input.Action == ActionSubmit || input.Action == ActionResubmit {
		if err = requireAttachmentForSubmission(ctx, tx, requestID, current); err != nil {
			return PurchaseRequest{}, err
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE purchase_requests
		SET status = $2::varchar,
			submitted_at = CASE
				WHEN $3::varchar IN ('SUBMIT', 'RESUBMIT') THEN now()
				ELSE submitted_at
			END,
			approved_at = CASE
				WHEN $2::varchar = 'APPROVED' THEN now()
				ELSE NULL
			END,
			sla_due_at = CASE
				WHEN $3::varchar IN ('SUBMIT', 'RESUBMIT') THEN
					now() + make_interval(hours => COALESCE((
						SELECT sp.target_hours
						FROM reporting.sla_policies sp
						WHERE sp.organization_id = $4
						  AND sp.process_name = 'PURCHASE_REQUEST_APPROVAL'
						  AND sp.active
					), 72))
				ELSE sla_due_at
			END,
			current_assignee_id = NULL,
			version = version + 1,
			updated_at = now()
		WHERE id = $1
	`, requestID, decision.ToStatus, input.Action, current.OrganizationID)
	if err != nil {
		return PurchaseRequest{}, fmt.Errorf("update purchase request transition: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO process_events (
			purchase_request_id,
			event_type,
			from_status,
			to_status,
			actor_id,
			actor_roles,
			comment,
			metadata,
			correlation_id,
			idempotency_key
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, NULLIF($7, ''),
			jsonb_build_object('action', $8::text),
			NULLIF($9, ''), $10
		)
	`, requestID, decision.EventType, current.Status, decision.ToStatus, user.ID,
		principal.Roles, input.Comment, input.Action, input.CorrelationID, input.IdempotencyKey)
	if err != nil {
		var databaseError *pgconn.PgError
		if errors.As(err, &databaseError) && databaseError.Code == "23505" {
			return PurchaseRequest{}, ErrIdempotencyConflict
		}
		return PurchaseRequest{}, fmt.Errorf("insert purchase request transition event: %w", err)
	}
	if err = insertAudit(
		ctx, tx, requestID, decision.EventType, user.ID, principal.Roles,
		string(current.Status), decision.ToStatus, input.CorrelationID,
	); err != nil {
		return PurchaseRequest{}, err
	}
	if err = applyBudgetTransition(
		ctx,
		tx,
		requestID,
		current,
		decision,
		user.ID,
		principal.Roles,
		input.CorrelationID,
	); err != nil {
		return PurchaseRequest{}, err
	}
	if err = queueTransitionNotification(ctx, tx, requestID, current, decision, user.ID); err != nil {
		return PurchaseRequest{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return PurchaseRequest{}, fmt.Errorf("commit purchase request transition: %w", err)
	}
	return s.getUnscoped(ctx, requestID)
}

func (s *Store) ensureUser(ctx context.Context, principal auth.Principal) (userProfile, error) {
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return userProfile{}, fmt.Errorf("begin user provisioning: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return userProfile{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return userProfile{}, fmt.Errorf("commit user provisioning: %w", err)
	}
	return user, nil
}

func ensureUser(
	ctx context.Context,
	tx pgx.Tx,
	principal auth.Principal,
) (userProfile, error) {
	user, err := identity.Ensure(ctx, tx, principal, defaultDepartmentCode)
	if errors.Is(err, identity.ErrInactive) {
		return userProfile{}, ErrForbidden
	}
	if err != nil {
		return userProfile{}, err
	}
	return user, nil
}

type lockedRequest struct {
	RequestCode    string
	Title          string
	RequesterID    string
	DepartmentID   string
	OrganizationID string
	Status         Status
	Version        int64
	CostCenter     string
	Currency       string
	TotalAmount    string
}

func canAccessLockedRequest(scope ScopeKind, user userProfile, request lockedRequest) bool {
	switch scope {
	case ScopeOwn:
		return request.RequesterID == user.ID
	case ScopeDepartment:
		return request.DepartmentID == user.DepartmentID
	case ScopeFinance:
		return request.OrganizationID == user.OrganizationID &&
			(request.Status == StatusManagerApproved ||
				request.Status == StatusApproved ||
				request.Status == StatusRejected)
	case ScopeAll:
		return true
	default:
		return false
	}
}

func lockRequest(ctx context.Context, tx pgx.Tx, requestID string) (lockedRequest, error) {
	var request lockedRequest
	err := tx.QueryRow(ctx, `
		SELECT
			pr.request_code,
			pr.title,
			pr.requester_id,
			pr.department_id,
			d.organization_id,
			pr.status,
			pr.version,
			pr.cost_center,
			pr.currency,
			pr.total_amount::text
		FROM purchase_requests pr
		JOIN departments d ON d.id = pr.department_id
		WHERE pr.id = $1
		FOR UPDATE OF pr
	`, requestID).Scan(
		&request.RequestCode,
		&request.Title,
		&request.RequesterID,
		&request.DepartmentID,
		&request.OrganizationID,
		&request.Status,
		&request.Version,
		&request.CostCenter,
		&request.Currency,
		&request.TotalAmount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return lockedRequest{}, ErrNotFound
	}
	if err != nil {
		return lockedRequest{}, fmt.Errorf("lock purchase request: %w", err)
	}
	return request, nil
}

func queueCommentNotification(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	request lockedRequest,
	actorID string,
) error {
	input := notifications.QueueInput{
		EventType: "COMMENT_ADDED", ResourceType: "purchase_request",
		ResourceID: requestID, OrganizationID: request.OrganizationID,
		DepartmentID: request.DepartmentID, ActorID: actorID,
		Title: "Phiếu mua sắm có bình luận mới",
		Body:  request.RequestCode + " - " + request.Title,
	}
	if actorID != request.RequesterID {
		input.RecipientUserID = request.RequesterID
	} else {
		switch request.Status {
		case StatusSubmitted:
			input.RecipientRole = "department_manager"
		case StatusManagerApproved:
			input.RecipientRole = "finance"
		default:
			return nil
		}
	}
	return notifications.Queue(ctx, tx, input)
}

func queueTransitionNotification(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	request lockedRequest,
	decision TransitionDecision,
	actorID string,
) error {
	input := notifications.QueueInput{
		EventType: decision.EventType, ResourceType: "purchase_request",
		ResourceID: requestID, OrganizationID: request.OrganizationID,
		DepartmentID: request.DepartmentID, ActorID: actorID,
		Body: request.RequestCode + " - " + request.Title,
	}
	switch decision.EventType {
	case "SUBMITTED", "RESUBMITTED":
		input.RecipientRole = "department_manager"
		input.Title = "Phiếu mua sắm cần phê duyệt"
	case "MANAGER_APPROVED":
		input.RecipientRole = "finance"
		input.DepartmentID = ""
		input.Title = "Phiếu chờ duyệt tài chính"
	case "FINANCE_APPROVED":
		input.RecipientUserID = request.RequesterID
		input.Title = "Phiếu mua sắm đã được phê duyệt"
	case "CHANGES_REQUESTED":
		input.RecipientUserID = request.RequesterID
		input.Title = "Phiếu mua sắm cần chỉnh sửa"
	case "REJECTED":
		input.RecipientUserID = request.RequesterID
		input.Title = "Phiếu mua sắm đã bị từ chối"
	default:
		return nil
	}
	return notifications.Queue(ctx, tx, input)
}

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func activeBudgetSummary(
	ctx context.Context,
	database queryRower,
	organizationID string,
	costCenter string,
	currency string,
) (BudgetSummary, error) {
	return scanBudgetSummary(database.QueryRow(ctx, `
		SELECT
			bp.code,
			bp.starts_on,
			bp.ends_on,
			ba.cost_center,
			ba.currency,
			ba.allocated_amount::text,
			ba.reserved_amount::text,
			ba.committed_amount::text,
			(ba.allocated_amount - ba.reserved_amount - ba.committed_amount)::text
		FROM budget_allocations ba
		JOIN budget_periods bp ON bp.id = ba.budget_period_id
		WHERE bp.organization_id = $1
		  AND bp.status = 'ACTIVE'
		  AND CURRENT_DATE BETWEEN bp.starts_on AND bp.ends_on
		  AND ba.cost_center = $2
		  AND ba.currency = $3
		ORDER BY bp.starts_on DESC
		LIMIT 1
	`, organizationID, costCenter, currency), nil)
}

func scanBudgetSummary(row pgx.Row, reservationState *string) (BudgetSummary, error) {
	var (
		summary                BudgetSummary
		periodStart, periodEnd time.Time
	)
	destinations := []any{
		&summary.PeriodCode,
		&periodStart,
		&periodEnd,
		&summary.CostCenter,
		&summary.Currency,
		&summary.AllocatedAmount,
		&summary.ReservedAmount,
		&summary.CommittedAmount,
		&summary.AvailableAmount,
	}
	if reservationState != nil {
		destinations = append(destinations, reservationState)
	}
	if err := row.Scan(destinations...); err != nil {
		return BudgetSummary{}, err
	}
	summary.PeriodStart = periodStart.Format(time.DateOnly)
	summary.PeriodEnd = periodEnd.Format(time.DateOnly)
	return summary, nil
}

func applyBudgetTransition(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	current lockedRequest,
	decision TransitionDecision,
	actorID string,
	actorRoles []string,
	correlationID string,
) error {
	switch {
	case current.Status == StatusSubmitted && decision.ToStatus == StatusManagerApproved:
		return reserveBudget(
			ctx, tx, requestID, current, decision.ToStatus,
			actorID, actorRoles, correlationID,
		)
	case current.Status == StatusManagerApproved && decision.ToStatus == StatusApproved:
		return settleBudgetReservation(
			ctx, tx, requestID, decision.ToStatus, "COMMITTED",
			actorID, actorRoles, correlationID,
		)
	case current.Status == StatusManagerApproved &&
		(decision.ToStatus == StatusRejected || decision.ToStatus == StatusChangesRequested):
		return settleBudgetReservation(
			ctx, tx, requestID, decision.ToStatus, "RELEASED",
			actorID, actorRoles, correlationID,
		)
	default:
		return nil
	}
}

func reserveBudget(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	request lockedRequest,
	toStatus Status,
	actorID string,
	actorRoles []string,
	correlationID string,
) error {
	var allocationID string
	var enough bool
	err := tx.QueryRow(ctx, `
		SELECT
			ba.id,
			$4::numeric <=
				(ba.allocated_amount - ba.reserved_amount - ba.committed_amount)
		FROM budget_allocations ba
		JOIN budget_periods bp ON bp.id = ba.budget_period_id
		WHERE bp.organization_id = $1
		  AND bp.status = 'ACTIVE'
		  AND CURRENT_DATE BETWEEN bp.starts_on AND bp.ends_on
		  AND ba.cost_center = $2
		  AND ba.currency = $3
		ORDER BY bp.starts_on DESC
		LIMIT 1
		FOR UPDATE OF ba
	`, request.OrganizationID, request.CostCenter, request.Currency, request.TotalAmount).Scan(
		&allocationID,
		&enough,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrBudgetNotConfigured
	}
	if err != nil {
		return fmt.Errorf("lock budget allocation: %w", err)
	}
	if !enough {
		return ErrInsufficientBudget
	}

	var reservationID string
	if err = tx.QueryRow(ctx, `
		INSERT INTO budget_reservations (
			budget_allocation_id,
			purchase_request_id,
			amount,
			currency,
			status,
			reserved_by
		)
		VALUES ($1, $2, $3::numeric, $4, 'RESERVED', $5)
		RETURNING id
	`, allocationID, requestID, request.TotalAmount, request.Currency, actorID).Scan(
		&reservationID,
	); err != nil {
		var databaseError *pgconn.PgError
		if errors.As(err, &databaseError) && databaseError.Code == "23505" {
			return ErrBudgetReservation
		}
		return fmt.Errorf("insert budget reservation: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		UPDATE budget_allocations
		SET reserved_amount = reserved_amount + $2::numeric,
			version = version + 1,
			updated_at = now()
		WHERE id = $1
	`, allocationID, request.TotalAmount); err != nil {
		return fmt.Errorf("increase reserved budget: %w", err)
	}
	return insertBudgetEvent(
		ctx, tx, requestID, "BUDGET_RESERVED", toStatus, actorID, actorRoles,
		correlationID, reservationID, request.TotalAmount, request.Currency,
	)
}

func settleBudgetReservation(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	toStatus Status,
	targetState string,
	actorID string,
	actorRoles []string,
	correlationID string,
) error {
	var reservationID, allocationID, amount, currency string
	err := tx.QueryRow(ctx, `
		SELECT br.id, ba.id, br.amount::text, br.currency
		FROM budget_reservations br
		JOIN budget_allocations ba ON ba.id = br.budget_allocation_id
		WHERE br.purchase_request_id = $1
		  AND br.status = 'RESERVED'
		FOR UPDATE OF br, ba
	`, requestID).Scan(&reservationID, &allocationID, &amount, &currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrBudgetReservation
	}
	if err != nil {
		return fmt.Errorf("lock budget reservation: %w", err)
	}

	eventType := "BUDGET_RELEASED"
	if targetState == "COMMITTED" {
		eventType = "BUDGET_COMMITTED"
		if _, err = tx.Exec(ctx, `
			UPDATE budget_allocations
			SET reserved_amount = reserved_amount - $2::numeric,
				committed_amount = committed_amount + $2::numeric,
				version = version + 1,
				updated_at = now()
			WHERE id = $1
		`, allocationID, amount); err != nil {
			return fmt.Errorf("commit reserved budget: %w", err)
		}
		if _, err = tx.Exec(ctx, `
			UPDATE budget_reservations
			SET status = 'COMMITTED',
				committed_by = $2,
				committed_at = now(),
				updated_at = now()
			WHERE id = $1
		`, reservationID, actorID); err != nil {
			return fmt.Errorf("mark budget reservation committed: %w", err)
		}
	} else {
		if _, err = tx.Exec(ctx, `
			UPDATE budget_allocations
			SET reserved_amount = reserved_amount - $2::numeric,
				version = version + 1,
				updated_at = now()
			WHERE id = $1
		`, allocationID, amount); err != nil {
			return fmt.Errorf("release reserved budget: %w", err)
		}
		if _, err = tx.Exec(ctx, `
			UPDATE budget_reservations
			SET status = 'RELEASED',
				released_by = $2,
				released_at = now(),
				updated_at = now()
			WHERE id = $1
		`, reservationID, actorID); err != nil {
			return fmt.Errorf("mark budget reservation released: %w", err)
		}
	}
	return insertBudgetEvent(
		ctx, tx, requestID, eventType, toStatus, actorID, actorRoles,
		correlationID, reservationID, amount, currency,
	)
}

func insertBudgetEvent(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	eventType string,
	status Status,
	actorID string,
	actorRoles []string,
	correlationID string,
	reservationID string,
	amount string,
	currency string,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO process_events (
			purchase_request_id,
			event_type,
			from_status,
			to_status,
			actor_id,
			actor_roles,
			metadata,
			correlation_id
		)
		VALUES (
			$1, $2, $3, $3, $4, $5,
			jsonb_build_object(
				'reservationId', $6::text,
				'amount', $7::text,
				'currency', $8::text
			),
			NULLIF($9, '')
		)
	`, requestID, eventType, status, actorID, actorRoles,
		reservationID, amount, currency, correlationID)
	if err != nil {
		return fmt.Errorf("insert %s process event: %w", strings.ToLower(eventType), err)
	}
	return insertAudit(
		ctx, tx, requestID, eventType, actorID, actorRoles,
		string(status), status, correlationID,
	)
}

func insertItems(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	items []CreateItemInput,
) error {
	for index, item := range items {
		_, err := tx.Exec(ctx, `
			INSERT INTO purchase_request_items (
				purchase_request_id,
				line_number,
				description,
				quantity,
				unit,
				unit_price
			)
			VALUES ($1, $2, $3, $4::numeric, $5, $6::numeric)
		`, requestID, index+1, item.Description, item.Quantity, item.Unit, item.UnitPrice)
		if err != nil {
			return fmt.Errorf("insert purchase request item %d: %w", index+1, err)
		}
	}
	return nil
}

func recalculateTotal(ctx context.Context, tx pgx.Tx, requestID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE purchase_requests
		SET total_amount = (
				SELECT COALESCE(SUM(line_total), 0)
				FROM purchase_request_items
				WHERE purchase_request_id = $1
			),
			updated_at = now()
		WHERE id = $1
	`, requestID)
	if err != nil {
		return fmt.Errorf("calculate purchase request total: %w", err)
	}
	return nil
}

func insertAudit(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	action string,
	actorID string,
	actorRoles []string,
	fromStatus string,
	toStatus Status,
	correlationID string,
) error {
	return insertResourceAudit(
		ctx, tx, "purchase_request", requestID, action, actorID, actorRoles,
		fromStatus, string(toStatus), correlationID,
	)
}

func insertResourceAudit(
	ctx context.Context,
	tx pgx.Tx,
	resourceType string,
	resourceID string,
	action string,
	actorID string,
	actorRoles []string,
	fromStatus string,
	toStatus string,
	correlationID string,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (
			resource_type,
			resource_id,
			action,
			actor_id,
			actor_roles,
			from_status,
			to_status,
			correlation_id,
			metadata
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			NULLIF($6, ''),
			NULLIF($7, ''),
			NULLIF($8, ''),
			jsonb_build_object('source', 'procurement')
		)
	`, resourceType, resourceID, action, actorID, actorRoles, fromStatus, toStatus, correlationID)
	if err != nil {
		return fmt.Errorf("insert %s audit: %w", resourceType, err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var databaseError *pgconn.PgError
	return errors.As(err, &databaseError) && databaseError.Code == "23505"
}

func (s *Store) getUnscoped(ctx context.Context, requestID string) (PurchaseRequest, error) {
	var request PurchaseRequest
	err := s.database.QueryRow(ctx, `
		SELECT
			pr.id,
			pr.request_code,
			pr.requester_id,
			u.display_name,
			pr.department_id,
			d.name,
			pr.title,
			pr.reason,
			pr.currency,
			pr.total_amount::text,
			pr.cost_center,
			pr.status,
			pr.version,
			pr.created_at,
			pr.updated_at
		FROM purchase_requests pr
		JOIN users u ON u.id = pr.requester_id
		JOIN departments d ON d.id = pr.department_id
		WHERE pr.id = $1
	`, requestID).Scan(
		&request.ID,
		&request.RequestCode,
		&request.RequesterID,
		&request.RequesterName,
		&request.DepartmentID,
		&request.DepartmentName,
		&request.Title,
		&request.Reason,
		&request.Currency,
		&request.TotalAmount,
		&request.CostCenter,
		&request.Status,
		&request.Version,
		&request.CreatedAt,
		&request.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PurchaseRequest{}, ErrNotFound
	}
	if err != nil {
		return PurchaseRequest{}, fmt.Errorf("get purchase request: %w", err)
	}

	rows, err := s.database.Query(ctx, `
		SELECT
			id,
			line_number,
			description,
			quantity::text,
			unit,
			unit_price::text,
			line_total::text
		FROM purchase_request_items
		WHERE purchase_request_id = $1
		ORDER BY line_number
	`, requestID)
	if err != nil {
		return PurchaseRequest{}, fmt.Errorf("list purchase request items: %w", err)
	}
	defer rows.Close()

	request.Items = make([]Item, 0)
	for rows.Next() {
		var item Item
		if err = rows.Scan(
			&item.ID,
			&item.LineNumber,
			&item.Description,
			&item.Quantity,
			&item.Unit,
			&item.UnitPrice,
			&item.LineTotal,
		); err != nil {
			return PurchaseRequest{}, fmt.Errorf("scan purchase request item: %w", err)
		}
		request.Items = append(request.Items, item)
	}
	if err = rows.Err(); err != nil {
		return PurchaseRequest{}, fmt.Errorf("iterate purchase request items: %w", err)
	}
	return request, nil
}
