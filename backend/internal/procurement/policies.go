package procurement

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/jackc/pgx/v5"
)

func (s *Store) PolicyCenter(ctx context.Context, principal auth.Principal) (PolicyCenter, error) {
	canManage := hasRole(principal.Roles, "dx_admin") && !hasRole(principal.Roles, "auditor")
	if !canManage && !hasRole(principal.Roles, "auditor") {
		return PolicyCenter{}, ErrForbidden
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return PolicyCenter{}, fmt.Errorf("begin policy center: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return PolicyCenter{}, err
	}
	result := PolicyCenter{SLA: []SLAPolicy{}, AttachmentRules: []AttachmentPolicy{}, CanManage: canManage}
	slaRows, err := tx.Query(ctx, `
		SELECT process_name, target_hours, active, version
		FROM reporting.sla_policies WHERE organization_id = $1 ORDER BY process_name
	`, user.OrganizationID)
	if err != nil {
		return PolicyCenter{}, fmt.Errorf("list SLA policies: %w", err)
	}
	for slaRows.Next() {
		var item SLAPolicy
		if err = slaRows.Scan(&item.ProcessName, &item.TargetHours, &item.Active, &item.Version); err != nil {
			slaRows.Close()
			return PolicyCenter{}, fmt.Errorf("scan SLA policy: %w", err)
		}
		result.SLA = append(result.SLA, item)
	}
	if err = slaRows.Err(); err != nil {
		slaRows.Close()
		return PolicyCenter{}, fmt.Errorf("iterate SLA policies: %w", err)
	}
	slaRows.Close()

	attachmentRows, err := tx.Query(ctx, `
		SELECT id, currency, threshold_amount::text, required_document_type, active, version
		FROM attachment_rules WHERE organization_id = $1 ORDER BY currency
	`, user.OrganizationID)
	if err != nil {
		return PolicyCenter{}, fmt.Errorf("list attachment policies: %w", err)
	}
	defer attachmentRows.Close()
	for attachmentRows.Next() {
		var item AttachmentPolicy
		if err = attachmentRows.Scan(&item.ID, &item.Currency, &item.ThresholdAmount, &item.RequiredDocumentType, &item.Active, &item.Version); err != nil {
			return PolicyCenter{}, fmt.Errorf("scan attachment policy: %w", err)
		}
		result.AttachmentRules = append(result.AttachmentRules, item)
	}
	if err = attachmentRows.Err(); err != nil {
		return PolicyCenter{}, fmt.Errorf("iterate attachment policies: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return PolicyCenter{}, fmt.Errorf("commit policy center: %w", err)
	}
	return result, nil
}

func (s *Store) UpdateSLAPolicy(ctx context.Context, principal auth.Principal, processName string, input UpdateSLAPolicyInput) (SLAPolicy, error) {
	if !hasRole(principal.Roles, "dx_admin") || hasRole(principal.Roles, "auditor") {
		return SLAPolicy{}, ErrForbidden
	}
	processName = strings.ToUpper(strings.TrimSpace(processName))
	violations := []FieldViolation{}
	if processName == "" || len(processName) > 80 {
		violations = append(violations, FieldViolation{Field: "processName", Message: "must contain 1 to 80 characters"})
	}
	if input.TargetHours < 1 || input.TargetHours > 720 {
		violations = append(violations, FieldViolation{Field: "targetHours", Message: "must be between 1 and 720"})
	}
	if input.ExpectedVersion < 1 {
		violations = append(violations, FieldViolation{Field: "expectedVersion", Message: "must be at least 1"})
	}
	if len(violations) > 0 {
		return SLAPolicy{}, &ValidationError{Violations: violations}
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return SLAPolicy{}, fmt.Errorf("begin SLA policy update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return SLAPolicy{}, err
	}
	var result SLAPolicy
	err = tx.QueryRow(ctx, `
		UPDATE reporting.sla_policies
		SET target_hours = $3, active = $4, version = version + 1, updated_by = $5, updated_at = now()
		WHERE organization_id = $1 AND process_name = $2 AND version = $6
		RETURNING process_name, target_hours, active, version
	`, user.OrganizationID, processName, input.TargetHours, input.Active, user.ID, input.ExpectedVersion).Scan(
		&result.ProcessName, &result.TargetHours, &result.Active, &result.Version,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return SLAPolicy{}, s.policyUpdateError(ctx, tx, "reporting.sla_policies", "process_name", processName, user.OrganizationID)
	}
	if err != nil {
		return SLAPolicy{}, fmt.Errorf("update SLA policy: %w", err)
	}
	if err = insertResourceAudit(ctx, tx, "operating_policy", user.OrganizationID, "SLA_POLICY_UPDATED", user.ID, principal.Roles, "", "", input.CorrelationID); err != nil {
		return SLAPolicy{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return SLAPolicy{}, fmt.Errorf("commit SLA policy update: %w", err)
	}
	return result, nil
}

func (s *Store) UpdateAttachmentPolicy(ctx context.Context, principal auth.Principal, ruleID string, input UpdateAttachmentPolicyInput) (AttachmentPolicy, error) {
	if !hasRole(principal.Roles, "dx_admin") || hasRole(principal.Roles, "auditor") {
		return AttachmentPolicy{}, ErrForbidden
	}
	input.ThresholdAmount = strings.TrimSpace(input.ThresholdAmount)
	input.RequiredDocumentType = strings.ToUpper(strings.TrimSpace(input.RequiredDocumentType))
	violations := []FieldViolation{}
	if number, valid := decimal(input.ThresholdAmount, unitPricePattern, false); !valid || number.Cmp(maxMoney) > 0 {
		violations = append(violations, FieldViolation{Field: "thresholdAmount", Message: "must be a non-negative supported decimal"})
	}
	if !slicesContains([]string{"QUOTATION", "SPECIFICATION", "CONTRACT", "OTHER"}, input.RequiredDocumentType) {
		violations = append(violations, FieldViolation{Field: "requiredDocumentType", Message: "must be QUOTATION, SPECIFICATION, CONTRACT, or OTHER"})
	}
	if input.ExpectedVersion < 1 {
		violations = append(violations, FieldViolation{Field: "expectedVersion", Message: "must be at least 1"})
	}
	if len(violations) > 0 {
		return AttachmentPolicy{}, &ValidationError{Violations: violations}
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return AttachmentPolicy{}, fmt.Errorf("begin attachment policy update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return AttachmentPolicy{}, err
	}
	var result AttachmentPolicy
	err = tx.QueryRow(ctx, `
		UPDATE attachment_rules
		SET threshold_amount = $3::numeric, required_document_type = $4, active = $5,
			version = version + 1, updated_by = $6, updated_at = now()
		WHERE organization_id = $1 AND id = $2 AND version = $7
		RETURNING id, currency, threshold_amount::text, required_document_type, active, version
	`, user.OrganizationID, ruleID, input.ThresholdAmount, input.RequiredDocumentType, input.Active, user.ID, input.ExpectedVersion).Scan(
		&result.ID, &result.Currency, &result.ThresholdAmount, &result.RequiredDocumentType, &result.Active, &result.Version,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return AttachmentPolicy{}, s.policyUpdateError(ctx, tx, "attachment_rules", "id", ruleID, user.OrganizationID)
	}
	if err != nil {
		return AttachmentPolicy{}, fmt.Errorf("update attachment policy: %w", err)
	}
	if err = insertResourceAudit(ctx, tx, "attachment_policy", result.ID, "ATTACHMENT_POLICY_UPDATED", user.ID, principal.Roles, "", "", input.CorrelationID); err != nil {
		return AttachmentPolicy{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return AttachmentPolicy{}, fmt.Errorf("commit attachment policy update: %w", err)
	}
	return result, nil
}

func (s *Store) policyUpdateError(ctx context.Context, tx pgx.Tx, table, keyColumn, key, organizationID string) error {
	query := "SELECT EXISTS (SELECT 1 FROM " + table + " WHERE organization_id = $1 AND " + keyColumn + " = $2)"
	var exists bool
	if err := tx.QueryRow(ctx, query, organizationID, key).Scan(&exists); err != nil {
		return fmt.Errorf("check policy version: %w", err)
	}
	if exists {
		return ErrPolicyVersion
	}
	return ErrPolicyNotFound
}

func slicesContains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
