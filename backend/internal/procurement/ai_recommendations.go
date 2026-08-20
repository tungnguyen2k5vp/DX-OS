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
	return !hasRole(principal.Roles, "auditor") &&
		(hasRole(principal.Roles, "ai_operator") || hasRole(principal.Roles, "dx_admin"))
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
	result := AIRecommendationList{Items: make([]AIRecommendation, 0), CanOperate: canOperateAI(principal), Methodology: "Khuyến nghị được tạo bằng các quy tắc kiểm soát có thể giải thích từ thời hạn xử lý, giá trị phiếu và rủi ro nhà cung cấp; hệ thống không tự động thay đổi dữ liệu nghiệp vụ."}
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
		 SELECT d.organization_id,pr.id,'SLA_BREACH_RISK','Phiếu quá hạn cần ưu tiên xử lý','Phiếu đang chờ xử lý đã quá thời hạn xử lý. Kiểm tra người phụ trách và nguyên nhân tắc nghẽn.','HIGH',jsonb_build_object('requestCode',pr.request_code,'status',pr.status,'slaDueAt',pr.sla_due_at),'SLA:'||pr.id::text||':'||pr.version::text,$2 FROM purchase_requests pr JOIN departments d ON d.id=pr.department_id WHERE d.organization_id=$1 AND pr.status IN ('SUBMITTED','MANAGER_APPROVED') AND pr.sla_due_at<now() ON CONFLICT(fingerprint) DO NOTHING`,
		`INSERT INTO ai_recommendations(organization_id,purchase_request_id,recommendation_type,title,summary,risk_level,evidence,fingerprint,generated_by)
		 SELECT d.organization_id,pr.id,'HIGH_VALUE_REVIEW','Rà soát phiếu giá trị lớn','Phiếu có giá trị từ 50 triệu VND; nên đối chiếu báo giá, ngân sách và điều kiện thương mại.','MEDIUM',jsonb_build_object('requestCode',pr.request_code,'amount',pr.total_amount,'currency',pr.currency),'VALUE:'||pr.id::text||':'||pr.version::text,$2 FROM purchase_requests pr JOIN departments d ON d.id=pr.department_id WHERE d.organization_id=$1 AND pr.currency='VND' AND pr.total_amount>=50000000 AND pr.status NOT IN ('REJECTED','CANCELLED') ON CONFLICT(fingerprint) DO NOTHING`,
		`INSERT INTO ai_recommendations(organization_id,purchase_request_id,recommendation_type,title,summary,risk_level,evidence,fingerprint,generated_by)
		 SELECT d.organization_id,pr.id,'SUPPLIER_RISK','Nhà cung cấp cần kiểm soát bổ sung','Đơn hàng liên quan nhà cung cấp có mức rủi ro cao hoặc trạng thái tuân thủ chưa đạt.','CRITICAL',jsonb_build_object('requestCode',pr.request_code,'supplierCode',s.code,'riskLevel',s.risk_level,'complianceStatus',s.compliance_status),'SUPPLIER:'||po.id::text||':'||s.version::text,$2 FROM purchase_orders po JOIN purchase_requests pr ON pr.id=po.purchase_request_id JOIN departments d ON d.id=pr.department_id JOIN suppliers s ON s.id=po.supplier_id WHERE d.organization_id=$1 AND (s.risk_level='HIGH' OR s.compliance_status IN ('EXPIRED','BLOCKED')) AND po.status<>'CANCELLED' ON CONFLICT(fingerprint) DO NOTHING`,
		`INSERT INTO ai_recommendations(organization_id,purchase_request_id,recommendation_type,title,summary,risk_level,evidence,fingerprint,generated_by)
		 SELECT d.organization_id,newer.id,'DUPLICATE_REQUEST_RISK','Có thể tạo phiếu mua sắm trùng','Có một phiếu gần đây trong cùng phòng ban có tiêu đề và giá trị tương tự. Hãy kiểm tra trước khi tiếp tục.','MEDIUM',jsonb_build_object('requestCode',newer.request_code,'matchingRequestCode',older.request_code,'amount',newer.total_amount,'matchingAmount',older.total_amount,'createdDaysApart',extract(day from newer.created_at-older.created_at)),'DUPREQ:'||newer.id::text||':'||newer.version::text||':'||older.id::text,$2 FROM purchase_requests newer JOIN purchase_requests older ON older.department_id=newer.department_id AND older.id<>newer.id AND lower(regexp_replace(older.title,'[^[:alnum:]]','','g'))=lower(regexp_replace(newer.title,'[^[:alnum:]]','','g')) AND older.created_at<newer.created_at AND older.created_at>=newer.created_at-interval '30 days' JOIN departments d ON d.id=newer.department_id WHERE d.organization_id=$1 AND newer.status NOT IN ('REJECTED','CANCELLED') AND older.status NOT IN ('REJECTED','CANCELLED') AND abs(newer.total_amount-older.total_amount)<=greatest(newer.total_amount,older.total_amount)*0.10 ON CONFLICT(fingerprint) DO NOTHING`,
		`INSERT INTO ai_recommendations(organization_id,purchase_request_id,recommendation_type,title,summary,risk_level,evidence,fingerprint,generated_by)
		 SELECT d.organization_id,pr.id,'SPLIT_PURCHASE_RISK','Có dấu hiệu chia nhỏ nhu cầu mua sắm','Nhiều phiếu nhỏ của cùng người yêu cầu và trung tâm chi phí trong 7 ngày có tổng giá trị vượt ngưỡng cần báo giá.','HIGH',jsonb_build_object('requestCode',pr.request_code,'requestAmount',pr.total_amount,'rollingSevenDayAmount',(SELECT sum(x.total_amount) FROM purchase_requests x WHERE x.requester_id=pr.requester_id AND x.cost_center=pr.cost_center AND x.currency=pr.currency AND x.created_at BETWEEN pr.created_at-interval '7 days' AND pr.created_at+interval '7 days' AND x.status NOT IN ('REJECTED','CANCELLED')),'threshold',20000000),'SPLIT:'||pr.id::text||':'||pr.version::text,$2 FROM purchase_requests pr JOIN departments d ON d.id=pr.department_id WHERE d.organization_id=$1 AND pr.currency='VND' AND pr.total_amount<20000000 AND pr.status NOT IN ('REJECTED','CANCELLED') AND (SELECT count(*) FROM purchase_requests x WHERE x.requester_id=pr.requester_id AND x.cost_center=pr.cost_center AND x.currency=pr.currency AND x.created_at BETWEEN pr.created_at-interval '7 days' AND pr.created_at+interval '7 days' AND x.status NOT IN ('REJECTED','CANCELLED'))>=2 AND (SELECT sum(x.total_amount) FROM purchase_requests x WHERE x.requester_id=pr.requester_id AND x.cost_center=pr.cost_center AND x.currency=pr.currency AND x.created_at BETWEEN pr.created_at-interval '7 days' AND pr.created_at+interval '7 days' AND x.status NOT IN ('REJECTED','CANCELLED'))>=20000000 ON CONFLICT(fingerprint) DO NOTHING`,
		`INSERT INTO ai_recommendations(organization_id,purchase_request_id,recommendation_type,title,summary,risk_level,evidence,fingerprint,generated_by)
		 SELECT DISTINCT ON (pr.id) d.organization_id,pr.id,'PRICE_ANOMALY','Đơn giá cao bất thường so với lịch sử','Một dòng hàng có đơn giá cao hơn đáng kể so với mức trung bình của các giao dịch cùng mô tả.','HIGH',jsonb_build_object('requestCode',pr.request_code,'itemDescription',pri.description,'unitPrice',pri.unit_price,'historicalAverage',history.average_price,'sampleSize',history.sample_size),'PRICE:'||pr.id::text||':'||pr.version::text,$2 FROM purchase_requests pr JOIN departments d ON d.id=pr.department_id JOIN purchase_request_items pri ON pri.purchase_request_id=pr.id JOIN LATERAL (SELECT avg(old_item.unit_price) average_price,count(*) sample_size FROM purchase_request_items old_item JOIN purchase_requests old_pr ON old_pr.id=old_item.purchase_request_id JOIN departments old_d ON old_d.id=old_pr.department_id WHERE old_d.organization_id=d.organization_id AND old_pr.id<>pr.id AND old_pr.status='APPROVED' AND lower(trim(old_item.description))=lower(trim(pri.description))) history ON history.sample_size>=2 WHERE d.organization_id=$1 AND pri.unit_price>history.average_price*1.35 AND pr.status NOT IN ('REJECTED','CANCELLED') ORDER BY pr.id,(pri.unit_price/history.average_price) DESC ON CONFLICT(fingerprint) DO NOTHING`,
		`INSERT INTO ai_recommendations(organization_id,purchase_request_id,recommendation_type,title,summary,risk_level,evidence,fingerprint,generated_by)
		 SELECT pi.organization_id,pr.id,'PAYMENT_OVERDUE','Hóa đơn đã quá hạn thanh toán','Hóa đơn đã xác minh nhưng vẫn còn số tiền chưa thanh toán sau ngày đến hạn.','HIGH',jsonb_build_object('requestCode',pr.request_code,'invoiceNumber',pi.invoice_number,'dueOn',pi.due_on,'invoiceAmount',pi.amount,'paidAmount',pi.paid_amount,'remainingAmount',pi.amount-pi.paid_amount),'PAYDUE:'||pi.id::text||':'||pi.version::text,$2 FROM purchase_invoices pi JOIN purchase_orders po ON po.id=pi.purchase_order_id JOIN purchase_requests pr ON pr.id=po.purchase_request_id WHERE pi.organization_id=$1 AND pi.status='VERIFIED' AND pi.due_on<CURRENT_DATE AND pi.paid_amount<pi.amount ON CONFLICT(fingerprint) DO NOTHING`,
		`INSERT INTO ai_recommendations(organization_id,purchase_request_id,recommendation_type,title,summary,risk_level,evidence,fingerprint,generated_by)
		 SELECT po.organization_id,pr.id,'SUPPLIER_MASTER_CHANGED','Thông tin nhà cung cấp thay đổi sau khi đặt hàng','Hồ sơ nhà cung cấp được cập nhật sau khi đơn hàng phát hành. Hãy kiểm tra lại thông tin thanh toán trước khi trả tiền.','CRITICAL',jsonb_build_object('requestCode',pr.request_code,'supplierCode',s.code,'supplierUpdatedAt',s.updated_at,'orderedAt',po.ordered_at,'bankAccountNumber',s.bank_account_number),'SUPCHANGE:'||po.id::text||':'||s.version::text,$2 FROM purchase_orders po JOIN purchase_requests pr ON pr.id=po.purchase_request_id JOIN suppliers s ON s.id=po.supplier_id WHERE po.organization_id=$1 AND po.status<>'CANCELLED' AND s.updated_at>po.ordered_at ON CONFLICT(fingerprint) DO NOTHING`,
		`INSERT INTO ai_recommendations(organization_id,purchase_request_id,recommendation_type,title,summary,risk_level,evidence,fingerprint,generated_by)
		 SELECT al.organization_id,pr.id,'ROLE_CONFLICT','Tài khoản có vai trò xung đột đã thao tác','Một tài khoản vừa mang vai trò tài chính vừa mang vai trò kiểm toán đã tác động lên giao dịch.','CRITICAL',jsonb_build_object('requestCode',pr.request_code,'actor',u.display_name,'roles',al.actor_roles,'action',al.action,'occurredAt',al.occurred_at),'ROLECONFLICT:'||al.id::text,$2 FROM audit_logs al JOIN purchase_requests pr ON al.resource_type='purchase_request' AND al.resource_id=pr.id JOIN users u ON u.id=al.actor_id WHERE al.organization_id=$1 AND al.actor_roles@>ARRAY['finance','auditor']::text[] ON CONFLICT(fingerprint) DO NOTHING`,
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
		violations = append(violations, FieldViolation{Field: "status", Message: "Phải là APPROVED (chấp thuận), REJECTED (từ chối) hoặc DISMISSED (bỏ qua)."})
	}
	if len([]rune(input.Comment)) < 5 || len([]rune(input.Comment)) > 2000 {
		violations = append(violations, FieldViolation{Field: "comment", Message: "Phải có từ 5 đến 2.000 ký tự."})
	}
	if input.ExpectedVersion < 1 {
		violations = append(violations, FieldViolation{Field: "expectedVersion", Message: "Phải lớn hơn 0."})
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
