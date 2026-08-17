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

type AIRecommendation struct {
	ID                   string          `json:"id"`
	PurchaseRequestID    *string         `json:"purchaseRequestId,omitempty"`
	PurchaseRequestCode  string          `json:"purchaseRequestCode,omitempty"`
	PurchaseRequestTitle string          `json:"purchaseRequestTitle,omitempty"`
	RecommendationType   string          `json:"recommendationType"`
	Title                string          `json:"title"`
	Summary              string          `json:"summary"`
	RiskLevel            string          `json:"riskLevel"`
	Evidence             json.RawMessage `json:"evidence"`
	Status               string          `json:"status"`
	GeneratedBy          string          `json:"generatedBy"`
	GeneratedAt          time.Time       `json:"generatedAt"`
	DecidedBy            string          `json:"decidedBy,omitempty"`
	DecidedAt            *time.Time      `json:"decidedAt,omitempty"`
	DecisionComment      string          `json:"decisionComment,omitempty"`
	Version              int64           `json:"version"`
}

type AIRecommendationList struct {
	Items       []AIRecommendation `json:"items"`
	Total       int                `json:"total"`
	Pending     int                `json:"pending"`
	HighRisk    int                `json:"highRisk"`
	CanOperate  bool               `json:"canOperate"`
	Methodology string             `json:"methodology"`
}

type DecideAIRecommendationInput struct {
	Status          string
	Comment         string
	ExpectedVersion int64
	CorrelationID   string
}

func canViewAI(principal auth.Principal) bool {
	return hasRole(principal.Roles, "ai_operator") || hasRole(principal.Roles, "dx_admin") || hasRole(principal.Roles, "finance") || hasRole(principal.Roles, "auditor")
}

func canOperateAI(principal auth.Principal) bool {
	return hasRole(principal.Roles, "ai_operator") || hasRole(principal.Roles, "dx_admin")
}

func (s *Store) ListAIRecommendations(ctx context.Context, principal auth.Principal) (AIRecommendationList, error) {
	if !canViewAI(principal) {
		return AIRecommendationList{}, ErrForbidden
	}
	user, err := s.ensureUser(ctx, principal)
	if err != nil {
		return AIRecommendationList{}, err
	}
	rows, err := s.database.Query(ctx, `
		SELECT ar.id,ar.purchase_request_id::text,COALESCE(pr.request_code,''),COALESCE(pr.title,''),ar.recommendation_type,ar.title,ar.summary,ar.risk_level,ar.evidence,ar.status,gu.display_name,ar.generated_at,COALESCE(du.display_name,''),ar.decided_at,COALESCE(ar.decision_comment,''),ar.version
		FROM ai_recommendations ar
		JOIN users gu ON gu.id=ar.generated_by
		LEFT JOIN users du ON du.id=ar.decided_by
		LEFT JOIN purchase_requests pr ON pr.id=ar.purchase_request_id
		WHERE ar.organization_id=$1
		ORDER BY CASE ar.status WHEN 'PENDING' THEN 1 ELSE 2 END,CASE ar.risk_level WHEN 'CRITICAL' THEN 1 WHEN 'HIGH' THEN 2 WHEN 'MEDIUM' THEN 3 ELSE 4 END,ar.generated_at DESC
	`, user.OrganizationID)
	if err != nil {
		return AIRecommendationList{}, fmt.Errorf("list AI recommendations: %w", err)
	}
	defer rows.Close()
	result := AIRecommendationList{Items: make([]AIRecommendation, 0), CanOperate: canOperateAI(principal), Methodology: "Khuyến nghị được tạo bằng các quy tắc kiểm soát có thể giải thích từ SLA, giá trị phiếu và rủi ro nhà cung cấp; không tự động thay đổi dữ liệu nghiệp vụ."}
	for rows.Next() {
		var item AIRecommendation
		if err = rows.Scan(&item.ID, &item.PurchaseRequestID, &item.PurchaseRequestCode, &item.PurchaseRequestTitle, &item.RecommendationType, &item.Title, &item.Summary, &item.RiskLevel, &item.Evidence, &item.Status, &item.GeneratedBy, &item.GeneratedAt, &item.DecidedBy, &item.DecidedAt, &item.DecisionComment, &item.Version); err != nil {
			return AIRecommendationList{}, err
		}
		result.Items = append(result.Items, item)
		if item.Status == "PENDING" {
			result.Pending++
		}
		if item.RiskLevel == "HIGH" || item.RiskLevel == "CRITICAL" {
			result.HighRisk++
		}
	}
	result.Total = len(result.Items)
	return result, rows.Err()
}

func (s *Store) GenerateAIRecommendations(ctx context.Context, principal auth.Principal) (AIRecommendationList, error) {
	if !canOperateAI(principal) {
		return AIRecommendationList{}, ErrForbidden
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return AIRecommendationList{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return AIRecommendationList{}, err
	}
	queries := []string{
		`INSERT INTO ai_recommendations(organization_id,purchase_request_id,recommendation_type,title,summary,risk_level,evidence,fingerprint,generated_by)
		 SELECT d.organization_id,pr.id,'SLA_BREACH_RISK','Phiếu quá hạn SLA cần ưu tiên','Phiếu đang chờ xử lý sau hạn SLA. Kiểm tra người phụ trách và nguyên nhân tắc nghẽn.','HIGH',jsonb_build_object('requestCode',pr.request_code,'status',pr.status,'slaDueAt',pr.sla_due_at),'SLA:'||pr.id::text||':'||pr.version::text,$2 FROM purchase_requests pr JOIN departments d ON d.id=pr.department_id WHERE d.organization_id=$1 AND pr.status IN ('SUBMITTED','MANAGER_APPROVED') AND pr.sla_due_at<now() ON CONFLICT(fingerprint) DO NOTHING`,
		`INSERT INTO ai_recommendations(organization_id,purchase_request_id,recommendation_type,title,summary,risk_level,evidence,fingerprint,generated_by)
		 SELECT d.organization_id,pr.id,'HIGH_VALUE_REVIEW','Rà soát phiếu giá trị lớn','Phiếu có giá trị từ 50 triệu VND; nên đối chiếu báo giá, ngân sách và điều kiện thương mại.','MEDIUM',jsonb_build_object('requestCode',pr.request_code,'amount',pr.total_amount,'currency',pr.currency),'VALUE:'||pr.id::text||':'||pr.version::text,$2 FROM purchase_requests pr JOIN departments d ON d.id=pr.department_id WHERE d.organization_id=$1 AND pr.currency='VND' AND pr.total_amount>=50000000 AND pr.status NOT IN ('REJECTED','CANCELLED') ON CONFLICT(fingerprint) DO NOTHING`,
		`INSERT INTO ai_recommendations(organization_id,purchase_request_id,recommendation_type,title,summary,risk_level,evidence,fingerprint,generated_by)
		 SELECT d.organization_id,pr.id,'SUPPLIER_RISK','Nhà cung cấp cần kiểm soát bổ sung','Đơn hàng liên quan nhà cung cấp có mức rủi ro cao hoặc trạng thái tuân thủ chưa đạt.','CRITICAL',jsonb_build_object('requestCode',pr.request_code,'supplierCode',s.code,'riskLevel',s.risk_level,'complianceStatus',s.compliance_status),'SUPPLIER:'||po.id::text||':'||s.version::text,$2 FROM purchase_orders po JOIN purchase_requests pr ON pr.id=po.purchase_request_id JOIN departments d ON d.id=pr.department_id JOIN suppliers s ON s.id=po.supplier_id WHERE d.organization_id=$1 AND (s.risk_level='HIGH' OR s.compliance_status IN ('EXPIRED','BLOCKED')) AND po.status<>'CANCELLED' ON CONFLICT(fingerprint) DO NOTHING`,
	}
	for _, query := range queries {
		if _, err = tx.Exec(ctx, query, user.OrganizationID, user.ID); err != nil {
			return AIRecommendationList{}, fmt.Errorf("generate recommendations: %w", err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return AIRecommendationList{}, err
	}
	return s.ListAIRecommendations(ctx, principal)
}

func (s *Store) DecideAIRecommendation(ctx context.Context, principal auth.Principal, id string, input DecideAIRecommendationInput) (AIRecommendation, error) {
	if !canOperateAI(principal) {
		return AIRecommendation{}, ErrForbidden
	}
	input.Status = strings.ToUpper(strings.TrimSpace(input.Status))
	input.Comment = strings.TrimSpace(input.Comment)
	var violations []FieldViolation
	if input.Status != "APPROVED" && input.Status != "REJECTED" && input.Status != "DISMISSED" {
		violations = append(violations, FieldViolation{Field: "status", Message: "must be APPROVED, REJECTED or DISMISSED"})
	}
	if len([]rune(input.Comment)) < 5 || len([]rune(input.Comment)) > 2000 {
		violations = append(violations, FieldViolation{Field: "comment", Message: "must contain between 5 and 2000 characters"})
	}
	if input.ExpectedVersion < 1 {
		violations = append(violations, FieldViolation{Field: "expectedVersion", Message: "must be greater than zero"})
	}
	if len(violations) > 0 {
		return AIRecommendation{}, &ValidationError{Violations: violations}
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return AIRecommendation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return AIRecommendation{}, err
	}
	var version int64
	var status string
	err = tx.QueryRow(ctx, `SELECT version,status FROM ai_recommendations WHERE id=$1 AND organization_id=$2 FOR UPDATE`, id, user.OrganizationID).Scan(&version, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return AIRecommendation{}, ErrAIRecommendationNotFound
	}
	if err != nil {
		return AIRecommendation{}, err
	}
	if version != input.ExpectedVersion {
		return AIRecommendation{}, ErrAIRecommendationVersion
	}
	if status != "PENDING" {
		return AIRecommendation{}, ErrInvalidAIAction
	}
	_, err = tx.Exec(ctx, `UPDATE ai_recommendations SET status=$2,decided_by=$3,decided_at=now(),decision_comment=$4,version=version+1 WHERE id=$1`, id, input.Status, user.ID, input.Comment)
	if err != nil {
		return AIRecommendation{}, err
	}
	if err = insertResourceAudit(ctx, tx, "ai_recommendation", id, "AI_RECOMMENDATION_DECIDED", user.ID, principal.Roles, status, input.Status, input.CorrelationID); err != nil {
		return AIRecommendation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return AIRecommendation{}, err
	}
	return s.getAIRecommendation(ctx, user.OrganizationID, id)
}

func (s *Store) getAIRecommendation(ctx context.Context, organizationID, id string) (AIRecommendation, error) {
	var item AIRecommendation
	err := s.database.QueryRow(ctx, `SELECT ar.id,ar.purchase_request_id::text,COALESCE(pr.request_code,''),COALESCE(pr.title,''),ar.recommendation_type,ar.title,ar.summary,ar.risk_level,ar.evidence,ar.status,gu.display_name,ar.generated_at,COALESCE(du.display_name,''),ar.decided_at,COALESCE(ar.decision_comment,''),ar.version FROM ai_recommendations ar JOIN users gu ON gu.id=ar.generated_by LEFT JOIN users du ON du.id=ar.decided_by LEFT JOIN purchase_requests pr ON pr.id=ar.purchase_request_id WHERE ar.id=$1 AND ar.organization_id=$2`, id, organizationID).Scan(&item.ID, &item.PurchaseRequestID, &item.PurchaseRequestCode, &item.PurchaseRequestTitle, &item.RecommendationType, &item.Title, &item.Summary, &item.RiskLevel, &item.Evidence, &item.Status, &item.GeneratedBy, &item.GeneratedAt, &item.DecidedBy, &item.DecidedAt, &item.DecisionComment, &item.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return AIRecommendation{}, ErrAIRecommendationNotFound
	}
	return item, err
}
