package procurement

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/jackc/pgx/v5"
)

type ApprovalRule struct {
	ID              string  `json:"id"`
	DepartmentID    *string `json:"departmentId,omitempty"`
	DepartmentName  string  `json:"departmentName,omitempty"`
	Name            string  `json:"name"`
	Currency        string  `json:"currency"`
	MinimumAmount   string  `json:"minimumAmount"`
	MaximumAmount   *string `json:"maximumAmount,omitempty"`
	RequiresManager bool    `json:"requiresManager"`
	RequiresFinance bool    `json:"requiresFinance"`
	Priority        int     `json:"priority"`
	Active          bool    `json:"active"`
	Version         int64   `json:"version"`
}

type ApprovalRuleInput struct {
	DepartmentID    string
	Name            string
	Currency        string
	MinimumAmount   string
	MaximumAmount   string
	RequiresManager bool
	RequiresFinance bool
	Priority        int
	Active          bool
	ExpectedVersion int64
	CorrelationID   string
}

type ApprovalDelegation struct {
	ID                 string   `json:"id"`
	DepartmentID       string   `json:"departmentId"`
	DepartmentName     string   `json:"departmentName"`
	DelegatorUserID    string   `json:"delegatorUserId"`
	DelegatorName      string   `json:"delegatorName"`
	DelegateUserID     string   `json:"delegateUserId"`
	DelegateName       string   `json:"delegateName"`
	DelegateRoles      []string `json:"delegateRoles"`
	StartsOn           string   `json:"startsOn"`
	EndsOn             string   `json:"endsOn"`
	Reason             string   `json:"reason"`
	Active             bool     `json:"active"`
	CurrentlyEffective bool     `json:"currentlyEffective"`
	Version            int64    `json:"version"`
}

type ApprovalDelegateCandidate struct {
	ID             string   `json:"id"`
	Username       string   `json:"username"`
	DisplayName    string   `json:"displayName"`
	DepartmentName string   `json:"departmentName"`
	Roles          []string `json:"roles"`
}

type ApprovalGovernance struct {
	Rules              []ApprovalRule              `json:"rules"`
	Delegations        []ApprovalDelegation        `json:"delegations"`
	DelegateCandidates []ApprovalDelegateCandidate `json:"delegateCandidates"`
	CanManageRules     bool                        `json:"canManageRules"`
	CanDelegate        bool                        `json:"canDelegate"`
}

type CreateDelegationInput struct {
	DelegateUserID string
	StartsOn       string
	EndsOn         string
	Reason         string
	CorrelationID  string
}

type SetDelegationActiveInput struct {
	Active          bool
	ExpectedVersion int64
	CorrelationID   string
}

type approvalRoute struct {
	RequiresManager bool
	RequiresFinance bool
	RuleID          string
	RuleName        string
}

func (s *Store) ApprovalGovernance(ctx context.Context, principal auth.Principal) (ApprovalGovernance, error) {
	canManage := hasRole(principal.Roles, "dx_admin") && !hasRole(principal.Roles, "auditor")
	canDelegate := hasRole(principal.Roles, "department_manager") && !hasRole(principal.Roles, "auditor")
	if !canManage && !canDelegate && !hasRole(principal.Roles, "finance") && !hasRole(principal.Roles, "auditor") {
		return ApprovalGovernance{}, ErrForbidden
	}
	user, err := s.ensureUser(ctx, principal)
	if err != nil {
		return ApprovalGovernance{}, err
	}
	result := ApprovalGovernance{
		Rules: make([]ApprovalRule, 0), Delegations: make([]ApprovalDelegation, 0),
		DelegateCandidates: make([]ApprovalDelegateCandidate, 0),
		CanManageRules:     canManage, CanDelegate: canDelegate,
	}
	ruleRows, err := s.database.Query(ctx, `
		SELECT ar.id,ar.department_id::text,COALESCE(d.name,''),ar.name,ar.currency,
		       ar.minimum_amount::text,ar.maximum_amount::text,ar.requires_manager,
		       ar.requires_finance,ar.priority,ar.active,ar.version
		FROM approval_rules ar
		LEFT JOIN departments d ON d.id=ar.department_id
		WHERE ar.organization_id=$1
		ORDER BY ar.active DESC,ar.priority,ar.minimum_amount
	`, user.OrganizationID)
	if err != nil {
		return ApprovalGovernance{}, fmt.Errorf("list approval rules: %w", err)
	}
	for ruleRows.Next() {
		var item ApprovalRule
		if err = ruleRows.Scan(&item.ID, &item.DepartmentID, &item.DepartmentName, &item.Name,
			&item.Currency, &item.MinimumAmount, &item.MaximumAmount, &item.RequiresManager,
			&item.RequiresFinance, &item.Priority, &item.Active, &item.Version); err != nil {
			ruleRows.Close()
			return ApprovalGovernance{}, err
		}
		result.Rules = append(result.Rules, item)
	}
	ruleRows.Close()
	delegationWhere := "ad.organization_id=$1"
	args := []any{user.OrganizationID}
	if canDelegate && !canManage && !hasRole(principal.Roles, "auditor") {
		delegationWhere += " AND (ad.delegator_user_id=$2 OR ad.delegate_user_id=$2)"
		args = append(args, user.ID)
	}
	delegationRows, err := s.database.Query(ctx, `
		SELECT ad.id,ad.department_id,d.name,ad.delegator_user_id,du.display_name,
		       ad.delegate_user_id,tu.display_name,COALESCE(urs.roles,'{}'),ad.starts_on::text,
		       ad.ends_on::text,ad.reason,ad.active,
		       (ad.active AND CURRENT_DATE BETWEEN ad.starts_on AND ad.ends_on),ad.version
		FROM approval_delegations ad
		JOIN departments d ON d.id=ad.department_id
		JOIN users du ON du.id=ad.delegator_user_id
		JOIN users tu ON tu.id=ad.delegate_user_id
		LEFT JOIN user_role_snapshots urs ON urs.user_id=tu.id
		WHERE `+delegationWhere+`
		ORDER BY ad.active DESC,ad.starts_on DESC,ad.created_at DESC
	`, args...)
	if err != nil {
		return ApprovalGovernance{}, fmt.Errorf("list approval delegations: %w", err)
	}
	for delegationRows.Next() {
		var item ApprovalDelegation
		if err = delegationRows.Scan(&item.ID, &item.DepartmentID, &item.DepartmentName,
			&item.DelegatorUserID, &item.DelegatorName, &item.DelegateUserID, &item.DelegateName,
			&item.DelegateRoles, &item.StartsOn, &item.EndsOn, &item.Reason, &item.Active,
			&item.CurrentlyEffective, &item.Version); err != nil {
			delegationRows.Close()
			return ApprovalGovernance{}, err
		}
		result.Delegations = append(result.Delegations, item)
	}
	delegationRows.Close()
	if canDelegate || canManage {
		candidateRows, queryErr := s.database.Query(ctx, `
			SELECT u.id,u.username,u.display_name,d.name,COALESCE(urs.roles,'{}')
			FROM users u
			JOIN departments d ON d.id=u.department_id
			LEFT JOIN user_role_snapshots urs ON urs.user_id=u.id
			WHERE d.organization_id=$1 AND u.active AND u.id<>$2
			ORDER BY u.display_name,u.username
		`, user.OrganizationID, user.ID)
		if queryErr != nil {
			return ApprovalGovernance{}, queryErr
		}
		for candidateRows.Next() {
			var item ApprovalDelegateCandidate
			if err = candidateRows.Scan(&item.ID, &item.Username, &item.DisplayName,
				&item.DepartmentName, &item.Roles); err != nil {
				candidateRows.Close()
				return ApprovalGovernance{}, err
			}
			result.DelegateCandidates = append(result.DelegateCandidates, item)
		}
		candidateRows.Close()
	}
	return result, nil
}

func (s *Store) CreateApprovalDelegation(ctx context.Context, principal auth.Principal, input CreateDelegationInput) (ApprovalDelegation, error) {
	if !hasRole(principal.Roles, "department_manager") || hasRole(principal.Roles, "auditor") {
		return ApprovalDelegation{}, ErrForbidden
	}
	input.DelegateUserID = strings.TrimSpace(input.DelegateUserID)
	input.Reason = strings.TrimSpace(input.Reason)
	startsOn, startErr := time.Parse(time.DateOnly, strings.TrimSpace(input.StartsOn))
	endsOn, endErr := time.Parse(time.DateOnly, strings.TrimSpace(input.EndsOn))
	var violations []FieldViolation
	if !uuidPatternForDomain.MatchString(input.DelegateUserID) {
		violations = append(violations, FieldViolation{Field: "delegateUserId", Message: "Người nhận ủy quyền không hợp lệ."})
	}
	if startErr != nil {
		violations = append(violations, FieldViolation{Field: "startsOn", Message: "Ngày bắt đầu phải có định dạng YYYY-MM-DD."})
	}
	if endErr != nil {
		violations = append(violations, FieldViolation{Field: "endsOn", Message: "Ngày kết thúc phải có định dạng YYYY-MM-DD."})
	} else if startErr == nil && endsOn.Before(startsOn) {
		violations = append(violations, FieldViolation{Field: "endsOn", Message: "Ngày kết thúc không được trước ngày bắt đầu."})
	}
	if len([]rune(input.Reason)) < 10 || len([]rune(input.Reason)) > 1000 {
		violations = append(violations, FieldViolation{Field: "reason", Message: "Lý do phải có từ 10 đến 1.000 ký tự."})
	}
	if len(violations) > 0 {
		return ApprovalDelegation{}, &ValidationError{Violations: violations}
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return ApprovalDelegation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	actor, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return ApprovalDelegation{}, err
	}
	if input.DelegateUserID == actor.ID {
		return ApprovalDelegation{}, ErrDelegationConflict
	}
	var targetOK bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users u JOIN departments d ON d.id=u.department_id WHERE u.id=$1 AND d.organization_id=$2 AND u.active)`, input.DelegateUserID, actor.OrganizationID).Scan(&targetOK); err != nil || !targetOK {
		return ApprovalDelegation{}, ErrDelegationConflict
	}
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO approval_delegations(organization_id,department_id,delegator_user_id,
			delegate_user_id,starts_on,ends_on,reason)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id
	`, actor.OrganizationID, actor.DepartmentID, actor.ID, input.DelegateUserID,
		startsOn, endsOn, input.Reason).Scan(&id)
	if err != nil {
		return ApprovalDelegation{}, fmt.Errorf("create approval delegation: %w", err)
	}
	if err = insertResourceAudit(ctx, tx, "approval_delegation", id, "APPROVAL_DELEGATION_CREATED",
		actor.ID, principal.Roles, "", "ACTIVE", input.CorrelationID); err != nil {
		return ApprovalDelegation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ApprovalDelegation{}, err
	}
	return s.getApprovalDelegation(ctx, actor.OrganizationID, id)
}

func (s *Store) SetApprovalDelegationActive(ctx context.Context, principal auth.Principal, id string, input SetDelegationActiveInput) (ApprovalDelegation, error) {
	if (!hasRole(principal.Roles, "department_manager") && !hasRole(principal.Roles, "dx_admin")) || hasRole(principal.Roles, "auditor") {
		return ApprovalDelegation{}, ErrForbidden
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return ApprovalDelegation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	actor, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return ApprovalDelegation{}, err
	}
	var version int64
	var delegatorID string
	err = tx.QueryRow(ctx, `SELECT version,delegator_user_id FROM approval_delegations WHERE id=$1 AND organization_id=$2 FOR UPDATE`, id, actor.OrganizationID).Scan(&version, &delegatorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ApprovalDelegation{}, ErrDelegationNotFound
	}
	if err != nil {
		return ApprovalDelegation{}, err
	}
	if !hasRole(principal.Roles, "dx_admin") && delegatorID != actor.ID {
		return ApprovalDelegation{}, ErrForbidden
	}
	if version != input.ExpectedVersion {
		return ApprovalDelegation{}, ErrDelegationVersion
	}
	_, err = tx.Exec(ctx, `UPDATE approval_delegations SET active=$2,version=version+1,updated_at=now() WHERE id=$1`, id, input.Active)
	if err != nil {
		return ApprovalDelegation{}, err
	}
	toStatus := "INACTIVE"
	if input.Active {
		toStatus = "ACTIVE"
	}
	if err = insertResourceAudit(ctx, tx, "approval_delegation", id, "APPROVAL_DELEGATION_STATUS_UPDATED",
		actor.ID, principal.Roles, "", toStatus, input.CorrelationID); err != nil {
		return ApprovalDelegation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ApprovalDelegation{}, err
	}
	return s.getApprovalDelegation(ctx, actor.OrganizationID, id)
}

func (s *Store) SaveApprovalRule(ctx context.Context, principal auth.Principal, id string, input ApprovalRuleInput) (ApprovalRule, error) {
	if !hasRole(principal.Roles, "dx_admin") || hasRole(principal.Roles, "auditor") {
		return ApprovalRule{}, ErrForbidden
	}
	if err := validateApprovalRule(&input, id != ""); err != nil {
		return ApprovalRule{}, err
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return ApprovalRule{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	actor, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return ApprovalRule{}, err
	}
	if input.DepartmentID != "" {
		var valid bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM departments WHERE id=$1 AND organization_id=$2)`, input.DepartmentID, actor.OrganizationID).Scan(&valid); err != nil || !valid {
			return ApprovalRule{}, ErrAdminDepartmentNotFound
		}
	}
	if id == "" {
		err = tx.QueryRow(ctx, `
			INSERT INTO approval_rules(organization_id,department_id,name,currency,minimum_amount,
				maximum_amount,requires_manager,requires_finance,priority,active,created_by,updated_by)
			VALUES($1,NULLIF($2,'')::uuid,$3,$4,$5::numeric,NULLIF($6,'')::numeric,$7,$8,$9,$10,$11,$11)
			RETURNING id
		`, actor.OrganizationID, input.DepartmentID, input.Name, input.Currency, input.MinimumAmount,
			input.MaximumAmount, input.RequiresManager, input.RequiresFinance, input.Priority,
			input.Active, actor.ID).Scan(&id)
	} else {
		var version int64
		err = tx.QueryRow(ctx, `SELECT version FROM approval_rules WHERE id=$1 AND organization_id=$2 FOR UPDATE`, id, actor.OrganizationID).Scan(&version)
		if errors.Is(err, pgx.ErrNoRows) {
			return ApprovalRule{}, ErrApprovalRuleNotFound
		}
		if err == nil && version != input.ExpectedVersion {
			return ApprovalRule{}, ErrApprovalRuleVersion
		}
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE approval_rules SET department_id=NULLIF($2,'')::uuid,name=$3,
				currency=$4,minimum_amount=$5::numeric,maximum_amount=NULLIF($6,'')::numeric,
				requires_manager=$7,requires_finance=$8,priority=$9,active=$10,updated_by=$11,
				version=version+1,updated_at=now() WHERE id=$1`, id, input.DepartmentID, input.Name,
				input.Currency, input.MinimumAmount, input.MaximumAmount, input.RequiresManager,
				input.RequiresFinance, input.Priority, input.Active, actor.ID)
		}
	}
	if err != nil {
		return ApprovalRule{}, fmt.Errorf("save approval rule: %w", err)
	}
	action := "APPROVAL_RULE_CREATED"
	if input.ExpectedVersion > 0 {
		action = "APPROVAL_RULE_UPDATED"
	}
	if err = insertResourceAudit(ctx, tx, "approval_rule", id, action, actor.ID, principal.Roles, "", "", input.CorrelationID); err != nil {
		return ApprovalRule{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ApprovalRule{}, err
	}
	return s.getApprovalRule(ctx, actor.OrganizationID, id)
}

func validateApprovalRule(input *ApprovalRuleInput, updating bool) error {
	input.DepartmentID = strings.TrimSpace(input.DepartmentID)
	input.Name = strings.TrimSpace(input.Name)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.MinimumAmount = strings.TrimSpace(input.MinimumAmount)
	input.MaximumAmount = strings.TrimSpace(input.MaximumAmount)
	var violations []FieldViolation
	if len([]rune(input.Name)) < 3 || len([]rune(input.Name)) > 255 {
		violations = append(violations, FieldViolation{Field: "name", Message: "Tên quy tắc phải có từ 3 đến 255 ký tự."})
	}
	if !currencyPattern.MatchString(input.Currency) {
		violations = append(violations, FieldViolation{Field: "currency", Message: "Tiền tệ phải gồm 3 chữ cái viết hoa."})
	}
	minimum, minimumOK := decimal(input.MinimumAmount, unitPricePattern, false)
	if !minimumOK {
		violations = append(violations, FieldViolation{Field: "minimumAmount", Message: "Giá trị tối thiểu không hợp lệ."})
	}
	if input.MaximumAmount != "" {
		maximum, maximumOK := decimal(input.MaximumAmount, unitPricePattern, false)
		if !maximumOK || (minimumOK && maximum.Cmp(minimum) < 0) {
			violations = append(violations, FieldViolation{Field: "maximumAmount", Message: "Giá trị tối đa phải lớn hơn hoặc bằng giá trị tối thiểu."})
		}
	}
	if !input.RequiresManager && !input.RequiresFinance {
		violations = append(violations, FieldViolation{Field: "requiresManager", Message: "Phải có ít nhất một cấp phê duyệt."})
	}
	if input.Priority < 0 || input.Priority > 100000 {
		violations = append(violations, FieldViolation{Field: "priority", Message: "Độ ưu tiên phải từ 0 đến 100.000."})
	}
	if input.DepartmentID != "" && !uuidPatternForDomain.MatchString(input.DepartmentID) {
		violations = append(violations, FieldViolation{Field: "departmentId", Message: "Phòng ban không hợp lệ."})
	}
	if updating && input.ExpectedVersion < 1 {
		violations = append(violations, FieldViolation{Field: "expectedVersion", Message: "Phiên bản phải lớn hơn 0."})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func (s *Store) resolveApprovalRoute(ctx context.Context, tx pgx.Tx, request lockedRequest) (approvalRoute, error) {
	var route approvalRoute
	err := tx.QueryRow(ctx, `
		SELECT id,name,requires_manager,requires_finance
		FROM approval_rules
		WHERE organization_id=$1 AND currency=$2 AND active
		  AND (department_id IS NULL OR department_id=$3)
		  AND minimum_amount <= $4::numeric
		  AND (maximum_amount IS NULL OR maximum_amount >= $4::numeric)
		ORDER BY (department_id IS NOT NULL) DESC,priority,minimum_amount DESC
		LIMIT 1
	`, request.OrganizationID, request.Currency, request.DepartmentID, request.TotalAmount).Scan(
		&route.RuleID, &route.RuleName, &route.RequiresManager, &route.RequiresFinance,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return approvalRoute{RequiresManager: true, RequiresFinance: true, RuleName: "Quy trình mặc định"}, nil
	}
	return route, err
}

func hasActiveApprovalDelegation(ctx context.Context, tx pgx.Tx, delegateUserID, departmentID, organizationID string) (bool, error) {
	var active bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM approval_delegations
		WHERE delegate_user_id=$1 AND department_id=$2 AND organization_id=$3 AND active
		  AND CURRENT_DATE BETWEEN starts_on AND ends_on
	)`, delegateUserID, departmentID, organizationID).Scan(&active)
	return active, err
}

func (s *Store) getApprovalDelegation(ctx context.Context, organizationID, id string) (ApprovalDelegation, error) {
	var item ApprovalDelegation
	err := s.database.QueryRow(ctx, `
		SELECT ad.id,ad.department_id,d.name,ad.delegator_user_id,du.display_name,
		       ad.delegate_user_id,tu.display_name,COALESCE(urs.roles,'{}'),ad.starts_on::text,
		       ad.ends_on::text,ad.reason,ad.active,
		       (ad.active AND CURRENT_DATE BETWEEN ad.starts_on AND ad.ends_on),ad.version
		FROM approval_delegations ad
		JOIN departments d ON d.id=ad.department_id
		JOIN users du ON du.id=ad.delegator_user_id
		JOIN users tu ON tu.id=ad.delegate_user_id
		LEFT JOIN user_role_snapshots urs ON urs.user_id=tu.id
		WHERE ad.id=$1 AND ad.organization_id=$2
	`, id, organizationID).Scan(&item.ID, &item.DepartmentID, &item.DepartmentName,
		&item.DelegatorUserID, &item.DelegatorName, &item.DelegateUserID, &item.DelegateName,
		&item.DelegateRoles, &item.StartsOn, &item.EndsOn, &item.Reason, &item.Active,
		&item.CurrentlyEffective, &item.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return ApprovalDelegation{}, ErrDelegationNotFound
	}
	return item, err
}

func (s *Store) getApprovalRule(ctx context.Context, organizationID, id string) (ApprovalRule, error) {
	var item ApprovalRule
	err := s.database.QueryRow(ctx, `SELECT ar.id,ar.department_id::text,COALESCE(d.name,''),ar.name,
		ar.currency,ar.minimum_amount::text,ar.maximum_amount::text,ar.requires_manager,
		ar.requires_finance,ar.priority,ar.active,ar.version FROM approval_rules ar
		LEFT JOIN departments d ON d.id=ar.department_id WHERE ar.id=$1 AND ar.organization_id=$2`,
		id, organizationID).Scan(&item.ID, &item.DepartmentID, &item.DepartmentName, &item.Name,
		&item.Currency, &item.MinimumAmount, &item.MaximumAmount, &item.RequiresManager,
		&item.RequiresFinance, &item.Priority, &item.Active, &item.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return ApprovalRule{}, ErrApprovalRuleNotFound
	}
	return item, err
}
