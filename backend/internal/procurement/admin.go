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

type AdminOverview struct {
	OrganizationName     string `json:"organizationName"`
	ActiveUsers          int    `json:"activeUsers"`
	InactiveUsers        int    `json:"inactiveUsers"`
	ActiveDepartments    int    `json:"activeDepartments"`
	OpenRequests         int    `json:"openRequests"`
	PendingNotifications int    `json:"pendingNotifications"`
	DeadNotifications    int    `json:"deadNotifications"`
}

type AdminUser struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	DisplayName    string    `json:"displayName"`
	DepartmentID   string    `json:"departmentId"`
	DepartmentName string    `json:"departmentName"`
	Active         bool      `json:"active"`
	Version        int64     `json:"version"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type AdminDepartment struct {
	ID         string  `json:"id"`
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	CostCenter string  `json:"costCenter"`
	ParentID   *string `json:"parentId,omitempty"`
	Active     bool    `json:"active"`
	Version    int64   `json:"version"`
}

type AdminCenter struct {
	Overview    AdminOverview     `json:"overview"`
	Users       []AdminUser       `json:"users"`
	Departments []AdminDepartment `json:"departments"`
	RoleNotice  string            `json:"roleNotice"`
}

type UpdateAdminUserInput struct {
	DisplayName     string
	Email           string
	DepartmentID    string
	Active          bool
	ExpectedVersion int64
	CorrelationID   string
}

type SaveDepartmentInput struct {
	Code            string
	Name            string
	CostCenter      string
	ParentID        string
	Active          bool
	ExpectedVersion int64
	CorrelationID   string
}

func (s *Store) AdminCenter(ctx context.Context, principal auth.Principal) (AdminCenter, error) {
	if !hasRole(principal.Roles, "dx_admin") {
		return AdminCenter{}, ErrForbidden
	}
	user, err := s.ensureUser(ctx, principal)
	if err != nil {
		return AdminCenter{}, err
	}
	result := AdminCenter{Users: make([]AdminUser, 0), Departments: make([]AdminDepartment, 0), RoleNotice: "Vai trò nghiệp vụ được quản lý tập trung trong Keycloak; màn hình này quản lý hồ sơ, phòng ban và trạng thái truy cập DX-OS."}
	err = s.database.QueryRow(ctx, `
		SELECT o.name,
			(SELECT count(*) FROM users u JOIN departments d ON d.id=u.department_id WHERE d.organization_id=o.id AND u.active),
			(SELECT count(*) FROM users u JOIN departments d ON d.id=u.department_id WHERE d.organization_id=o.id AND NOT u.active),
			(SELECT count(*) FROM departments d WHERE d.organization_id=o.id AND d.active),
			(SELECT count(*) FROM purchase_requests pr JOIN departments d ON d.id=pr.department_id WHERE d.organization_id=o.id AND pr.status NOT IN ('APPROVED','REJECTED','CANCELLED')),
			(SELECT count(*) FROM outbox_events oe WHERE oe.organization_id=o.id AND oe.status='PENDING'),
			(SELECT count(*) FROM outbox_events oe WHERE oe.organization_id=o.id AND oe.status='DEAD')
		FROM organizations o WHERE o.id=$1
	`, user.OrganizationID).Scan(&result.Overview.OrganizationName, &result.Overview.ActiveUsers, &result.Overview.InactiveUsers, &result.Overview.ActiveDepartments, &result.Overview.OpenRequests, &result.Overview.PendingNotifications, &result.Overview.DeadNotifications)
	if err != nil {
		return AdminCenter{}, fmt.Errorf("load admin overview: %w", err)
	}
	rows, err := s.database.Query(ctx, `SELECT u.id,u.username,COALESCE(u.email,''),u.display_name,d.id,d.name,u.active,u.version,u.updated_at FROM users u JOIN departments d ON d.id=u.department_id WHERE d.organization_id=$1 ORDER BY u.active DESC,u.display_name,u.username`, user.OrganizationID)
	if err != nil {
		return AdminCenter{}, fmt.Errorf("list admin users: %w", err)
	}
	for rows.Next() {
		var item AdminUser
		if err = rows.Scan(&item.ID, &item.Username, &item.Email, &item.DisplayName, &item.DepartmentID, &item.DepartmentName, &item.Active, &item.Version, &item.UpdatedAt); err != nil {
			rows.Close()
			return AdminCenter{}, err
		}
		result.Users = append(result.Users, item)
	}
	rows.Close()
	rows, err = s.database.Query(ctx, `SELECT id,code,name,COALESCE(cost_center,''),parent_id::text,active,version FROM departments WHERE organization_id=$1 ORDER BY active DESC,code`, user.OrganizationID)
	if err != nil {
		return AdminCenter{}, fmt.Errorf("list admin departments: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item AdminDepartment
		if err = rows.Scan(&item.ID, &item.Code, &item.Name, &item.CostCenter, &item.ParentID, &item.Active, &item.Version); err != nil {
			return AdminCenter{}, err
		}
		result.Departments = append(result.Departments, item)
	}
	return result, rows.Err()
}

func validateAdminUser(input *UpdateAdminUserInput) error {
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Email = strings.TrimSpace(input.Email)
	input.DepartmentID = strings.TrimSpace(input.DepartmentID)
	var violations []FieldViolation
	if len([]rune(input.DisplayName)) < 2 || len([]rune(input.DisplayName)) > 255 {
		violations = append(violations, FieldViolation{Field: "displayName", Message: "must contain between 2 and 255 characters"})
	}
	if input.Email != "" && !emailPattern.MatchString(input.Email) {
		violations = append(violations, FieldViolation{Field: "email", Message: "must be a valid email address"})
	}
	if !uuidPatternForDomain.MatchString(input.DepartmentID) {
		violations = append(violations, FieldViolation{Field: "departmentId", Message: "must be a valid UUID"})
	}
	if input.ExpectedVersion < 1 {
		violations = append(violations, FieldViolation{Field: "expectedVersion", Message: "must be greater than zero"})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func (s *Store) UpdateAdminUser(ctx context.Context, principal auth.Principal, targetID string, input UpdateAdminUserInput) (AdminUser, error) {
	if !hasRole(principal.Roles, "dx_admin") {
		return AdminUser{}, ErrForbidden
	}
	if err := validateAdminUser(&input); err != nil {
		return AdminUser{}, err
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return AdminUser{}, fmt.Errorf("begin admin user update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	actor, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return AdminUser{}, err
	}
	var version int64
	var currentActive bool
	err = tx.QueryRow(ctx, `SELECT u.version,u.active FROM users u JOIN departments d ON d.id=u.department_id WHERE u.id=$1 AND d.organization_id=$2 FOR UPDATE OF u`, targetID, actor.OrganizationID).Scan(&version, &currentActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminUser{}, ErrAdminUserNotFound
	}
	if err != nil {
		return AdminUser{}, fmt.Errorf("lock admin user: %w", err)
	}
	if version != input.ExpectedVersion {
		return AdminUser{}, ErrAdminVersion
	}
	if targetID == actor.ID && !input.Active {
		return AdminUser{}, ErrAdminConflict
	}
	var departmentOK bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM departments WHERE id=$1 AND organization_id=$2 AND active)`, input.DepartmentID, actor.OrganizationID).Scan(&departmentOK)
	if err != nil || !departmentOK {
		return AdminUser{}, ErrAdminDepartmentNotFound
	}
	_, err = tx.Exec(ctx, `UPDATE users SET display_name=$2,email=NULLIF($3,''),department_id=$4,active=$5,version=version+1,updated_at=now() WHERE id=$1`, targetID, input.DisplayName, input.Email, input.DepartmentID, input.Active)
	if err != nil {
		return AdminUser{}, fmt.Errorf("update admin user: %w", err)
	}
	if err = insertResourceAudit(ctx, tx, "user", targetID, "USER_ADMIN_UPDATED", actor.ID, principal.Roles, fmt.Sprint(currentActive), fmt.Sprint(input.Active), input.CorrelationID); err != nil {
		return AdminUser{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return AdminUser{}, err
	}
	return s.getAdminUser(ctx, actor.OrganizationID, targetID)
}

func (s *Store) getAdminUser(ctx context.Context, organizationID, targetID string) (AdminUser, error) {
	var item AdminUser
	err := s.database.QueryRow(ctx, `SELECT u.id,u.username,COALESCE(u.email,''),u.display_name,d.id,d.name,u.active,u.version,u.updated_at FROM users u JOIN departments d ON d.id=u.department_id WHERE u.id=$1 AND d.organization_id=$2`, targetID, organizationID).Scan(&item.ID, &item.Username, &item.Email, &item.DisplayName, &item.DepartmentID, &item.DepartmentName, &item.Active, &item.Version, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminUser{}, ErrAdminUserNotFound
	}
	return item, err
}

func validateDepartment(input *SaveDepartmentInput, updating bool) error {
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.CostCenter = strings.TrimSpace(input.CostCenter)
	input.ParentID = strings.TrimSpace(input.ParentID)
	var violations []FieldViolation
	if len(input.Code) < 2 || len(input.Code) > 50 || !supplierCodePattern.MatchString(input.Code) {
		violations = append(violations, FieldViolation{Field: "code", Message: "must be 2-50 uppercase letters, digits, dot, dash or underscore"})
	}
	if len([]rune(input.Name)) < 2 || len([]rune(input.Name)) > 255 {
		violations = append(violations, FieldViolation{Field: "name", Message: "must contain between 2 and 255 characters"})
	}
	if len(input.CostCenter) > 100 {
		violations = append(violations, FieldViolation{Field: "costCenter", Message: "must not exceed 100 characters"})
	}
	if input.ParentID != "" && !uuidPatternForDomain.MatchString(input.ParentID) {
		violations = append(violations, FieldViolation{Field: "parentId", Message: "must be a valid UUID"})
	}
	if updating && input.ExpectedVersion < 1 {
		violations = append(violations, FieldViolation{Field: "expectedVersion", Message: "must be greater than zero"})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func (s *Store) CreateDepartment(ctx context.Context, principal auth.Principal, input SaveDepartmentInput) (AdminDepartment, error) {
	if !hasRole(principal.Roles, "dx_admin") {
		return AdminDepartment{}, ErrForbidden
	}
	if err := validateDepartment(&input, false); err != nil {
		return AdminDepartment{}, err
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return AdminDepartment{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	actor, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return AdminDepartment{}, err
	}
	if err = validateDepartmentParent(ctx, tx, actor.OrganizationID, "", input.ParentID); err != nil {
		return AdminDepartment{}, err
	}
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO departments(organization_id,parent_id,code,name,cost_center,active) VALUES($1,NULLIF($2,'')::uuid,$3,$4,NULLIF($5,''),$6) RETURNING id`, actor.OrganizationID, input.ParentID, input.Code, input.Name, input.CostCenter, input.Active).Scan(&id)
	if isUniqueViolation(err) {
		return AdminDepartment{}, ErrAdminConflict
	}
	if err != nil {
		return AdminDepartment{}, fmt.Errorf("create department: %w", err)
	}
	if err = insertResourceAudit(ctx, tx, "department", id, "DEPARTMENT_CREATED", actor.ID, principal.Roles, "", fmt.Sprint(input.Active), input.CorrelationID); err != nil {
		return AdminDepartment{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return AdminDepartment{}, err
	}
	return s.getDepartment(ctx, actor.OrganizationID, id)
}

func (s *Store) UpdateDepartment(ctx context.Context, principal auth.Principal, id string, input SaveDepartmentInput) (AdminDepartment, error) {
	if !hasRole(principal.Roles, "dx_admin") {
		return AdminDepartment{}, ErrForbidden
	}
	if err := validateDepartment(&input, true); err != nil {
		return AdminDepartment{}, err
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return AdminDepartment{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	actor, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return AdminDepartment{}, err
	}
	var version int64
	err = tx.QueryRow(ctx, `SELECT version FROM departments WHERE id=$1 AND organization_id=$2 FOR UPDATE`, id, actor.OrganizationID).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminDepartment{}, ErrAdminDepartmentNotFound
	}
	if err != nil {
		return AdminDepartment{}, err
	}
	if version != input.ExpectedVersion {
		return AdminDepartment{}, ErrAdminVersion
	}
	if err = validateDepartmentParent(ctx, tx, actor.OrganizationID, id, input.ParentID); err != nil {
		return AdminDepartment{}, err
	}
	if !input.Active {
		var hasActiveUsers bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE department_id=$1 AND active)`, id).Scan(&hasActiveUsers); err != nil {
			return AdminDepartment{}, err
		}
		if hasActiveUsers {
			return AdminDepartment{}, ErrAdminConflict
		}
	}
	_, err = tx.Exec(ctx, `UPDATE departments SET parent_id=NULLIF($2,'')::uuid,code=$3,name=$4,cost_center=NULLIF($5,''),active=$6,version=version+1,updated_at=now() WHERE id=$1`, id, input.ParentID, input.Code, input.Name, input.CostCenter, input.Active)
	if isUniqueViolation(err) {
		return AdminDepartment{}, ErrAdminConflict
	}
	if err != nil {
		return AdminDepartment{}, err
	}
	if err = insertResourceAudit(ctx, tx, "department", id, "DEPARTMENT_UPDATED", actor.ID, principal.Roles, "", fmt.Sprint(input.Active), input.CorrelationID); err != nil {
		return AdminDepartment{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return AdminDepartment{}, err
	}
	return s.getDepartment(ctx, actor.OrganizationID, id)
}

func validateDepartmentParent(ctx context.Context, tx pgx.Tx, organizationID, departmentID, parentID string) error {
	if parentID == "" {
		return nil
	}
	var valid bool
	if departmentID == "" {
		err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM departments WHERE id=$1 AND organization_id=$2 AND active)`, parentID, organizationID).Scan(&valid)
		if err != nil {
			return err
		}
		if !valid {
			return ErrAdminConflict
		}
		return nil
	}
	err := tx.QueryRow(ctx, `
		WITH RECURSIVE descendants AS (
			SELECT id FROM departments WHERE id=$1
			UNION ALL SELECT d.id FROM departments d JOIN descendants p ON d.parent_id=p.id
		)
		SELECT EXISTS(
			SELECT 1 FROM departments p
			WHERE p.id=$2 AND p.organization_id=$3 AND p.active
				AND p.id NOT IN (SELECT id FROM descendants)
		)
	`, departmentID, parentID, organizationID).Scan(&valid)
	if err != nil {
		return err
	}
	if !valid {
		return ErrAdminConflict
	}
	return nil
}

func (s *Store) getDepartment(ctx context.Context, organizationID, id string) (AdminDepartment, error) {
	var item AdminDepartment
	err := s.database.QueryRow(ctx, `SELECT id,code,name,COALESCE(cost_center,''),parent_id::text,active,version FROM departments WHERE id=$1 AND organization_id=$2`, id, organizationID).Scan(&item.ID, &item.Code, &item.Name, &item.CostCenter, &item.ParentID, &item.Active, &item.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminDepartment{}, ErrAdminDepartmentNotFound
	}
	return item, err
}
