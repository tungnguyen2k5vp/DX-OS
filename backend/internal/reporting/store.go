package reporting

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	database *pgxpool.Pool
}

func NewStore(database *pgxpool.Pool) *Store {
	return &Store{database: database}
}

func (s *Store) Dashboard(
	ctx context.Context,
	principal auth.Principal,
	input DashboardInput,
) (Dashboard, error) {
	if !CanAccess(principal) {
		return Dashboard{}, ErrForbidden
	}
	if err := ValidateDashboardInput(&input); err != nil {
		return Dashboard{}, err
	}

	var organizationID string
	if isFinanceOnly(principal) {
		var err error
		organizationID, err = s.ensureOrganizationScope(ctx, principal)
		if err != nil {
			return Dashboard{}, err
		}
	}

	result := Dashboard{
		Filters: AppliedFilters{
			From:         input.From.Format(time.DateOnly),
			To:           input.To.Format(time.DateOnly),
			DepartmentID: input.DepartmentID,
			CostCenter:   input.CostCenter,
			Currency:     input.Currency,
		},
		CurrencyTotals: make([]CurrencyTotal, 0),
		Statuses:       make([]StatusBreakdown, 0),
		Trends:         make([]DailyTrend, 0),
		Departments:    make([]DepartmentBreakdown, 0),
		Budgets:        make([]BudgetUtilization, 0),
		GeneratedAt:    time.Now().UTC(),
	}

	where, args := factFilters(input, organizationID)
	if err := s.database.QueryRow(ctx, `
		SELECT
			count(*),
			count(*) FILTER (WHERE status = 'APPROVED'),
			count(*) FILTER (WHERE status = 'REJECTED'),
			count(*) FILTER (WHERE returned_for_changes),
			count(*) FILTER (WHERE sla_breached),
			COALESCE(round(avg(lead_time_hours), 2), 0)::text,
			count(*) FILTER (WHERE attachment_required),
			count(*) FILTER (WHERE attachment_required AND attachment_compliant),
			CASE
				WHEN count(*) FILTER (WHERE attachment_required) = 0 THEN '100.00'
				ELSE round(
					count(*) FILTER (WHERE attachment_required AND attachment_compliant)::numeric
					/ count(*) FILTER (WHERE attachment_required)::numeric * 100,
					2
				)::text
			END
		FROM reporting.purchase_request_facts
	`+where, args...).Scan(
		&result.Summary.TotalRequests,
		&result.Summary.ApprovedCount,
		&result.Summary.RejectedCount,
		&result.Summary.ReturnedCount,
		&result.Summary.SLABreachedCount,
		&result.Summary.AverageLeadTimeHours,
		&result.Summary.AttachmentRequiredCount,
		&result.Summary.AttachmentCompliantCount,
		&result.Summary.AttachmentComplianceRate,
	); err != nil {
		return Dashboard{}, fmt.Errorf("query reporting summary: %w", err)
	}

	if err := s.loadCurrencyTotals(ctx, &result, where, args); err != nil {
		return Dashboard{}, err
	}
	if err := s.loadStatuses(ctx, &result, where, args); err != nil {
		return Dashboard{}, err
	}
	if err := s.loadTrends(ctx, &result, input, organizationID); err != nil {
		return Dashboard{}, err
	}
	if err := s.loadDepartments(ctx, &result, where, args); err != nil {
		return Dashboard{}, err
	}
	if err := s.loadBudgets(ctx, &result, input, organizationID); err != nil {
		return Dashboard{}, err
	}
	return result, nil
}

func (s *Store) AuditCenter(
	ctx context.Context,
	principal auth.Principal,
	input AuditInput,
) (AuditCenter, error) {
	if !CanAccessAudit(principal) {
		return AuditCenter{}, ErrForbidden
	}
	if err := ValidateAuditInput(&input); err != nil {
		return AuditCenter{}, err
	}
	var conditions []string
	var args []any
	add := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if input.ResourceType != "" {
		conditions = append(conditions, "al.resource_type = "+add(input.ResourceType))
	}
	if input.Action != "" {
		conditions = append(conditions, "al.action = "+add(input.Action))
	}
	if input.From != nil {
		conditions = append(conditions, "al.occurred_at >= "+add(*input.From))
	}
	if input.To != nil {
		conditions = append(conditions, "al.occurred_at < "+add(*input.To))
	}
	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	limit := add(input.PageSize)
	offset := add((input.Page - 1) * input.PageSize)
	rows, err := s.database.Query(ctx, `
		SELECT
			al.id, al.resource_type, al.resource_id, al.action, u.display_name,
			al.actor_roles, al.from_status, al.to_status, al.correlation_id,
			al.occurred_at, count(*) OVER()
		FROM audit_logs al
		JOIN users u ON u.id = al.actor_id
		`+where+`
		ORDER BY al.occurred_at DESC, al.id DESC
		LIMIT `+limit+` OFFSET `+offset,
		args...,
	)
	if err != nil {
		return AuditCenter{}, fmt.Errorf("query audit center: %w", err)
	}
	defer rows.Close()
	result := AuditCenter{Items: make([]AuditEvent, 0), Page: input.Page, PageSize: input.PageSize}
	for rows.Next() {
		var event AuditEvent
		if err = rows.Scan(
			&event.ID, &event.ResourceType, &event.ResourceID, &event.Action,
			&event.ActorName, &event.ActorRoles, &event.FromStatus, &event.ToStatus,
			&event.CorrelationID, &event.OccurredAt, &result.Total,
		); err != nil {
			return AuditCenter{}, fmt.Errorf("scan audit center event: %w", err)
		}
		result.Items = append(result.Items, event)
	}
	if err = rows.Err(); err != nil {
		return AuditCenter{}, fmt.Errorf("iterate audit center events: %w", err)
	}
	if result.Total > 0 {
		result.Pages = int((result.Total + int64(input.PageSize) - 1) / int64(input.PageSize))
	}
	if err = s.database.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE occurred_at >= CURRENT_DATE),
			count(*) FILTER (WHERE resource_type = 'supplier'),
			count(*) FILTER (WHERE resource_type = 'purchase_order')
		FROM audit_logs
	`).Scan(
		&result.TodayCount,
		&result.SupplierChangeCount,
		&result.PurchaseOrderEventCount,
	); err != nil {
		return AuditCenter{}, fmt.Errorf("query audit center summary: %w", err)
	}
	return result, nil
}

func (s *Store) loadCurrencyTotals(
	ctx context.Context,
	result *Dashboard,
	where string,
	args []any,
) error {
	rows, err := s.database.Query(ctx, `
		SELECT currency, count(*), COALESCE(sum(total_amount), 0)::text
		FROM reporting.purchase_request_facts
	`+where+`
		GROUP BY currency
		ORDER BY currency
	`, args...)
	if err != nil {
		return fmt.Errorf("query reporting currency totals: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item CurrencyTotal
		if err = rows.Scan(&item.Currency, &item.RequestCount, &item.TotalAmount); err != nil {
			return fmt.Errorf("scan reporting currency total: %w", err)
		}
		result.CurrencyTotals = append(result.CurrencyTotals, item)
	}
	return rows.Err()
}

func (s *Store) loadStatuses(
	ctx context.Context,
	result *Dashboard,
	where string,
	args []any,
) error {
	rows, err := s.database.Query(ctx, `
		SELECT status, currency, count(*), COALESCE(sum(total_amount), 0)::text
		FROM reporting.purchase_request_facts
	`+where+`
		GROUP BY status, currency
		ORDER BY
			CASE status
				WHEN 'DRAFT' THEN 1
				WHEN 'SUBMITTED' THEN 2
				WHEN 'MANAGER_APPROVED' THEN 3
				WHEN 'CHANGES_REQUESTED' THEN 4
				WHEN 'APPROVED' THEN 5
				WHEN 'REJECTED' THEN 6
				ELSE 7
			END,
			currency
	`, args...)
	if err != nil {
		return fmt.Errorf("query reporting statuses: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item StatusBreakdown
		if err = rows.Scan(&item.Status, &item.Currency, &item.RequestCount, &item.TotalAmount); err != nil {
			return fmt.Errorf("scan reporting status: %w", err)
		}
		result.Statuses = append(result.Statuses, item)
	}
	return rows.Err()
}

func (s *Store) loadTrends(
	ctx context.Context,
	result *Dashboard,
	input DashboardInput,
	organizationID string,
) error {
	where, args := metricFilters(input, organizationID)
	rows, err := s.database.Query(ctx, `
		SELECT
			metric_date,
			currency,
			sum(request_count),
			sum(approved_count),
			COALESCE(sum(total_amount), 0)::text
		FROM reporting.daily_procurement_metrics
	`+where+`
		GROUP BY metric_date, currency
		ORDER BY metric_date, currency
	`, args...)
	if err != nil {
		return fmt.Errorf("query reporting trends: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item DailyTrend
		var date time.Time
		if err = rows.Scan(
			&date, &item.Currency, &item.RequestCount, &item.ApprovedCount, &item.TotalAmount,
		); err != nil {
			return fmt.Errorf("scan reporting trend: %w", err)
		}
		item.Date = date.Format(time.DateOnly)
		result.Trends = append(result.Trends, item)
	}
	return rows.Err()
}

func (s *Store) loadDepartments(
	ctx context.Context,
	result *Dashboard,
	where string,
	args []any,
) error {
	rows, err := s.database.Query(ctx, `
		SELECT
			department_id,
			department_name,
			currency,
			count(*),
			count(*) FILTER (WHERE status = 'APPROVED'),
			COALESCE(sum(total_amount), 0)::text
		FROM reporting.purchase_request_facts
	`+where+`
		GROUP BY department_id, department_name, currency
		ORDER BY count(*) DESC, department_name, currency
	`, args...)
	if err != nil {
		return fmt.Errorf("query reporting departments: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item DepartmentBreakdown
		if err = rows.Scan(
			&item.DepartmentID,
			&item.DepartmentName,
			&item.Currency,
			&item.RequestCount,
			&item.ApprovedCount,
			&item.TotalAmount,
		); err != nil {
			return fmt.Errorf("scan reporting department: %w", err)
		}
		result.Departments = append(result.Departments, item)
	}
	return rows.Err()
}

func (s *Store) loadBudgets(
	ctx context.Context,
	result *Dashboard,
	input DashboardInput,
	organizationID string,
) error {
	var conditions []string
	var args []any
	add := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	conditions = append(conditions, "period_status = 'ACTIVE'")
	if organizationID != "" {
		conditions = append(conditions, "organization_id = "+add(organizationID)+"::uuid")
	}
	if input.CostCenter != "" {
		conditions = append(conditions, "cost_center = "+add(input.CostCenter))
	}
	if input.Currency != "" {
		conditions = append(conditions, "currency = "+add(input.Currency))
	}
	rows, err := s.database.Query(ctx, `
		SELECT
			period_code,
			period_start,
			period_end,
			cost_center,
			currency,
			allocated_amount::text,
			reserved_amount::text,
			committed_amount::text,
			available_amount::text,
			utilization_percent::text
		FROM reporting.budget_utilization
		WHERE `+strings.Join(conditions, " AND ")+`
		ORDER BY period_start DESC, cost_center, currency
	`, args...)
	if err != nil {
		return fmt.Errorf("query reporting budgets: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item BudgetUtilization
		var start, end time.Time
		if err = rows.Scan(
			&item.PeriodCode,
			&start,
			&end,
			&item.CostCenter,
			&item.Currency,
			&item.AllocatedAmount,
			&item.ReservedAmount,
			&item.CommittedAmount,
			&item.AvailableAmount,
			&item.UtilizationPercent,
		); err != nil {
			return fmt.Errorf("scan reporting budget: %w", err)
		}
		item.PeriodStart = start.Format(time.DateOnly)
		item.PeriodEnd = end.Format(time.DateOnly)
		result.Budgets = append(result.Budgets, item)
	}
	return rows.Err()
}

func factFilters(input DashboardInput, organizationID string) (string, []any) {
	var conditions []string
	var args []any
	add := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	conditions = append(
		conditions,
		"created_date >= "+add(input.From)+"::date",
		"created_date <= "+add(input.To)+"::date",
	)
	if organizationID != "" {
		conditions = append(conditions, "organization_id = "+add(organizationID)+"::uuid")
	}
	if input.DepartmentID != "" {
		conditions = append(conditions, "department_id = "+add(input.DepartmentID)+"::uuid")
	}
	if input.CostCenter != "" {
		conditions = append(conditions, "cost_center = "+add(input.CostCenter))
	}
	if input.Currency != "" {
		conditions = append(conditions, "currency = "+add(input.Currency))
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func metricFilters(input DashboardInput, organizationID string) (string, []any) {
	var conditions []string
	var args []any
	add := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	conditions = append(
		conditions,
		"metric_date >= "+add(input.From)+"::date",
		"metric_date <= "+add(input.To)+"::date",
	)
	if organizationID != "" {
		conditions = append(conditions, "organization_id = "+add(organizationID)+"::uuid")
	}
	if input.DepartmentID != "" {
		conditions = append(conditions, "department_id = "+add(input.DepartmentID)+"::uuid")
	}
	if input.CostCenter != "" {
		conditions = append(conditions, "cost_center = "+add(input.CostCenter))
	}
	if input.Currency != "" {
		conditions = append(conditions, "currency = "+add(input.Currency))
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func (s *Store) ensureOrganizationScope(
	ctx context.Context,
	principal auth.Principal,
) (string, error) {
	var organizationID string
	var active bool
	err := s.database.QueryRow(ctx, `
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
		WHERE d.code = 'GENERAL' AND d.active
		ORDER BY d.created_at
		LIMIT 1
		ON CONFLICT (keycloak_subject) DO UPDATE
		SET username = EXCLUDED.username,
			email = COALESCE(EXCLUDED.email, users.email),
			display_name = EXCLUDED.display_name,
			updated_at = now()
		RETURNING
			(SELECT organization_id FROM departments WHERE id = users.department_id),
			users.active
	`, principal.Subject, principal.Username, principal.Email).Scan(&organizationID, &active)
	if err == pgx.ErrNoRows {
		return "", ErrForbidden
	}
	if err != nil {
		return "", fmt.Errorf("resolve reporting organization: %w", err)
	}
	if !active {
		return "", ErrForbidden
	}
	return organizationID, nil
}
