package procurement

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultDepartmentCode = "GENERAL"

type userProfile struct {
	ID             string
	DepartmentID   string
	OrganizationID string
	Active         bool
}

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

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	limitPlaceholder := addArgument(input.PageSize)
	offsetPlaceholder := addArgument((input.Page - 1) * input.PageSize)

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
		ORDER BY pr.created_at DESC, pr.id DESC
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
			current_assignee_id = NULL,
			version = version + 1,
			updated_at = now()
		WHERE id = $1
	`, requestID, decision.ToStatus, input.Action)
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
	var user userProfile
	err := tx.QueryRow(ctx, `
		INSERT INTO users (
			keycloak_subject,
			username,
			email,
			display_name,
			department_id
		)
		SELECT
			$1,
			$2,
			NULLIF($3, ''),
			$2,
			d.id
		FROM departments d
		WHERE d.code = $4 AND d.active
		ORDER BY d.created_at
		LIMIT 1
		ON CONFLICT (keycloak_subject) DO UPDATE
		SET username = EXCLUDED.username,
			email = COALESCE(EXCLUDED.email, users.email),
			display_name = EXCLUDED.display_name,
			updated_at = now()
		RETURNING
			users.id,
			users.department_id,
			(SELECT organization_id FROM departments WHERE id = users.department_id),
			users.active
	`, principal.Subject, principal.Username, principal.Email, defaultDepartmentCode).Scan(
		&user.ID,
		&user.DepartmentID,
		&user.OrganizationID,
		&user.Active,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return userProfile{}, errors.New("default active department is not configured")
	}
	if err != nil {
		return userProfile{}, fmt.Errorf("provision business user: %w", err)
	}
	if !user.Active {
		return userProfile{}, ErrForbidden
	}
	return user, nil
}

type lockedRequest struct {
	RequesterID    string
	DepartmentID   string
	OrganizationID string
	Status         Status
	Version        int64
	CostCenter     string
	Currency       string
	TotalAmount    string
}

func lockRequest(ctx context.Context, tx pgx.Tx, requestID string) (lockedRequest, error) {
	var request lockedRequest
	err := tx.QueryRow(ctx, `
		SELECT
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
			'purchase_request',
			$1,
			$2,
			$3,
			$4,
			NULLIF($5, ''),
			$6,
			NULLIF($7, ''),
			jsonb_build_object('source', 'procurement')
		)
	`, requestID, action, actorID, actorRoles, fromStatus, toStatus, correlationID)
	if err != nil {
		return fmt.Errorf("insert purchase request audit: %w", err)
	}
	return nil
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
