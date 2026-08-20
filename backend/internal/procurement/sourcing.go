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
)

type SupplierQuote struct {
	ID              string  `json:"id"`
	SourcingCaseID  string  `json:"sourcingCaseId"`
	SupplierID      string  `json:"supplierId"`
	SupplierCode    string  `json:"supplierCode"`
	SupplierName    string  `json:"supplierName"`
	QuoteReference  string  `json:"quoteReference"`
	Amount          string  `json:"amount"`
	Currency        string  `json:"currency"`
	DeliveryOn      string  `json:"deliveryOn"`
	WarrantyMonths  int     `json:"warrantyMonths"`
	PaymentTerms    string  `json:"paymentTerms"`
	Note            string  `json:"note,omitempty"`
	Status          string  `json:"status"`
	PriceScore      float64 `json:"priceScore"`
	DeliveryScore   float64 `json:"deliveryScore"`
	QualityScore    float64 `json:"qualityScore"`
	ComplianceScore float64 `json:"complianceScore"`
	TotalScore      float64 `json:"totalScore"`
	Recommendation  string  `json:"recommendation"`
	Version         int64   `json:"version"`
}

type SourcingCase struct {
	ID                 *string         `json:"id,omitempty"`
	PurchaseRequestID  string          `json:"purchaseRequestId"`
	RequestCode        string          `json:"requestCode"`
	RequestTitle       string          `json:"requestTitle"`
	DepartmentName     string          `json:"departmentName"`
	RequesterName      string          `json:"requesterName"`
	RequestAmount      string          `json:"requestAmount"`
	Currency           string          `json:"currency"`
	Status             string          `json:"status"`
	SelectedQuoteID    *string         `json:"selectedQuoteId,omitempty"`
	Quotes             []SupplierQuote `json:"quotes"`
	RecommendedQuoteID *string         `json:"recommendedQuoteId,omitempty"`
	PotentialSavings   string          `json:"potentialSavings"`
	CanManage          bool            `json:"canManage"`
	Version            int64           `json:"version"`
}

type SourcingBoard struct {
	Items          []SourcingCase `json:"items"`
	Total          int            `json:"total"`
	AwaitingQuotes int            `json:"awaitingQuotes"`
	InComparison   int            `json:"inComparison"`
	Awarded        int            `json:"awarded"`
	CanManage      bool           `json:"canManage"`
}

type SupplierQuoteInput struct {
	PurchaseRequestID string
	SupplierID        string
	QuoteReference    string
	Amount            string
	Currency          string
	DeliveryOn        string
	WarrantyMonths    int
	PaymentTerms      string
	Note              string
	IdempotencyKey    string
	CorrelationID     string
}

type UpdateSupplierQuoteInput struct {
	QuoteReference  string
	Amount          string
	Currency        string
	DeliveryOn      string
	WarrantyMonths  int
	PaymentTerms    string
	Note            string
	ExpectedVersion int64
	CorrelationID   string
}

type SelectSupplierQuoteInput struct {
	ExpectedCaseVersion  int64
	ExpectedQuoteVersion int64
	Comment              string
	IdempotencyKey       string
	CorrelationID        string
}

func canViewSourcing(principal auth.Principal) bool {
	return hasRole(principal.Roles, "finance") || hasRole(principal.Roles, "auditor") || hasRole(principal.Roles, "dx_admin")
}

func canManageSourcing(principal auth.Principal) bool {
	return hasRole(principal.Roles, "finance") && !hasRole(principal.Roles, "auditor")
}

func (s *Store) SourcingBoard(ctx context.Context, principal auth.Principal) (SourcingBoard, error) {
	if !canViewSourcing(principal) {
		return SourcingBoard{}, ErrForbidden
	}
	user, err := s.ensureUser(ctx, principal)
	if err != nil {
		return SourcingBoard{}, err
	}
	rows, err := s.database.Query(ctx, `
		SELECT pr.id,pr.request_code,pr.title,d.name,u.display_name,pr.total_amount::text,
		       pr.currency,sc.id::text,COALESCE(sc.status,'NOT_STARTED'),sc.selected_quote_id::text,
		       COALESCE(sc.version,0)
		FROM purchase_requests pr
		JOIN departments d ON d.id=pr.department_id
		JOIN users u ON u.id=pr.requester_id
		LEFT JOIN sourcing_cases sc ON sc.purchase_request_id=pr.id
		WHERE d.organization_id=$1 AND pr.status='APPROVED'
		  AND NOT EXISTS (
		    SELECT 1 FROM purchase_orders po
		    WHERE po.purchase_request_id=pr.id AND po.status<>'CANCELLED'
		  )
		ORDER BY CASE COALESCE(sc.status,'NOT_STARTED') WHEN 'OPEN' THEN 1 WHEN 'NOT_STARTED' THEN 2 ELSE 3 END,
		         pr.approved_at DESC,pr.created_at DESC
	`, user.OrganizationID)
	if err != nil {
		return SourcingBoard{}, fmt.Errorf("list sourcing cases: %w", err)
	}
	defer rows.Close()
	result := SourcingBoard{Items: make([]SourcingCase, 0), CanManage: canManageSourcing(principal)}
	for rows.Next() {
		var item SourcingCase
		if err = rows.Scan(&item.PurchaseRequestID, &item.RequestCode, &item.RequestTitle,
			&item.DepartmentName, &item.RequesterName, &item.RequestAmount, &item.Currency,
			&item.ID, &item.Status, &item.SelectedQuoteID, &item.Version); err != nil {
			return SourcingBoard{}, err
		}
		item.CanManage = result.CanManage
		item.Quotes = make([]SupplierQuote, 0)
		item.PotentialSavings = "0"
		if item.ID != nil {
			quotes, quoteErr := s.listSupplierQuotes(ctx, *item.ID)
			if quoteErr != nil {
				return SourcingBoard{}, quoteErr
			}
			item.Quotes = quotes
			if len(quotes) > 0 {
				best := quotes[0]
				for _, quote := range quotes[1:] {
					if quote.TotalScore > best.TotalScore {
						best = quote
					}
				}
				item.RecommendedQuoteID = &best.ID
				if savings, ok := subtractMoney(item.RequestAmount, best.Amount); ok {
					item.PotentialSavings = savings
				}
			}
		}
		switch {
		case item.Status == "AWARDED":
			result.Awarded++
		case len(item.Quotes) == 0:
			result.AwaitingQuotes++
		default:
			result.InComparison++
		}
		result.Items = append(result.Items, item)
	}
	result.Total = len(result.Items)
	return result, rows.Err()
}

func (s *Store) CreateSupplierQuote(ctx context.Context, principal auth.Principal, input SupplierQuoteInput) (SupplierQuote, error) {
	if !canManageSourcing(principal) {
		return SupplierQuote{}, ErrForbidden
	}
	if err := validateSupplierQuote(&input.QuoteReference, &input.Amount, &input.Currency,
		&input.DeliveryOn, input.WarrantyMonths, &input.PaymentTerms, &input.Note); err != nil {
		return SupplierQuote{}, err
	}
	input.PurchaseRequestID = strings.TrimSpace(input.PurchaseRequestID)
	input.SupplierID = strings.TrimSpace(input.SupplierID)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if !uuidPatternForDomain.MatchString(input.PurchaseRequestID) || !uuidPatternForDomain.MatchString(input.SupplierID) {
		return SupplierQuote{}, &ValidationError{Violations: []FieldViolation{{Field: "purchaseRequestId", Message: "Phiếu hoặc nhà cung cấp không hợp lệ."}}}
	}
	if !idempotencyPattern.MatchString(input.IdempotencyKey) {
		return SupplierQuote{}, &ValidationError{Violations: []FieldViolation{{Field: "Idempotency-Key", Message: "Khóa chống gửi trùng không hợp lệ."}}}
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return SupplierQuote{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return SupplierQuote{}, err
	}
	var existingQuoteID string
	err = tx.QueryRow(ctx, `SELECT quote_id::text FROM sourcing_events WHERE idempotency_key=$1`, input.IdempotencyKey).Scan(&existingQuoteID)
	if err == nil {
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return SupplierQuote{}, commitErr
		}
		return s.getSupplierQuote(ctx, existingQuoteID)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return SupplierQuote{}, err
	}
	var organizationID, requestCurrency, status string
	err = tx.QueryRow(ctx, `SELECT d.organization_id,pr.currency,pr.status FROM purchase_requests pr JOIN departments d ON d.id=pr.department_id WHERE pr.id=$1 FOR UPDATE OF pr`, input.PurchaseRequestID).Scan(&organizationID, &requestCurrency, &status)
	if errors.Is(err, pgx.ErrNoRows) || organizationID != user.OrganizationID {
		return SupplierQuote{}, ErrNotFound
	}
	if err != nil {
		return SupplierQuote{}, err
	}
	if status != string(StatusApproved) || input.Currency != requestCurrency {
		return SupplierQuote{}, ErrInvalidSourcingAction
	}
	var supplierOK bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM suppliers WHERE id=$1 AND organization_id=$2 AND status='ACTIVE' AND compliance_status<>'BLOCKED')`, input.SupplierID, user.OrganizationID).Scan(&supplierOK)
	if err != nil || !supplierOK {
		return SupplierQuote{}, ErrSupplierNotFound
	}
	var caseID, caseStatus string
	err = tx.QueryRow(ctx, `
		INSERT INTO sourcing_cases(organization_id,purchase_request_id,created_by)
		VALUES($1,$2,$3) ON CONFLICT(purchase_request_id) DO UPDATE SET updated_at=now()
		RETURNING id,status
	`, user.OrganizationID, input.PurchaseRequestID, user.ID).Scan(&caseID, &caseStatus)
	if err != nil {
		return SupplierQuote{}, err
	}
	if caseStatus != "OPEN" {
		return SupplierQuote{}, ErrInvalidSourcingAction
	}
	var quoteID string
	err = tx.QueryRow(ctx, `
		INSERT INTO supplier_quotes(sourcing_case_id,supplier_id,quote_reference,amount,currency,
			delivery_on,warranty_months,payment_terms,note,created_by)
		VALUES($1,$2,$3,$4::numeric,$5,$6::date,$7,$8,NULLIF($9,''),$10) RETURNING id
	`, caseID, input.SupplierID, input.QuoteReference, input.Amount, input.Currency,
		input.DeliveryOn, input.WarrantyMonths, input.PaymentTerms, input.Note, user.ID).Scan(&quoteID)
	if err != nil {
		var databaseError *pgconn.PgError
		if errors.As(err, &databaseError) && databaseError.Code == "23505" {
			return SupplierQuote{}, ErrIdempotencyConflict
		}
		return SupplierQuote{}, fmt.Errorf("create supplier quote: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO sourcing_events(sourcing_case_id,quote_id,event_type,actor_id,actor_roles,comment,correlation_id,idempotency_key) VALUES($1,$2,'QUOTE_RECORDED',$3,$4,NULLIF($5,''),NULLIF($6,''),$7)`, caseID, quoteID, user.ID, principal.Roles, input.Note, input.CorrelationID, input.IdempotencyKey)
	if err != nil {
		return SupplierQuote{}, err
	}
	if err = insertResourceAudit(ctx, tx, "supplier_quote", quoteID, "SUPPLIER_QUOTE_RECORDED", user.ID, principal.Roles, "", "SUBMITTED", input.CorrelationID); err != nil {
		return SupplierQuote{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return SupplierQuote{}, err
	}
	return s.getSupplierQuote(ctx, quoteID)
}

func (s *Store) UpdateSupplierQuote(ctx context.Context, principal auth.Principal, quoteID string, input UpdateSupplierQuoteInput) (SupplierQuote, error) {
	if !canManageSourcing(principal) {
		return SupplierQuote{}, ErrForbidden
	}
	if err := validateSupplierQuote(&input.QuoteReference, &input.Amount, &input.Currency,
		&input.DeliveryOn, input.WarrantyMonths, &input.PaymentTerms, &input.Note); err != nil {
		return SupplierQuote{}, err
	}
	if input.ExpectedVersion < 1 {
		return SupplierQuote{}, &ValidationError{Violations: []FieldViolation{{Field: "expectedVersion", Message: "Phiên bản phải lớn hơn 0."}}}
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return SupplierQuote{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return SupplierQuote{}, err
	}
	var version int64
	var status, organizationID, requestCurrency string
	err = tx.QueryRow(ctx, `SELECT q.version,q.status,sc.organization_id,pr.currency FROM supplier_quotes q JOIN sourcing_cases sc ON sc.id=q.sourcing_case_id JOIN purchase_requests pr ON pr.id=sc.purchase_request_id WHERE q.id=$1 FOR UPDATE OF q`, quoteID).Scan(&version, &status, &organizationID, &requestCurrency)
	if errors.Is(err, pgx.ErrNoRows) || organizationID != user.OrganizationID {
		return SupplierQuote{}, ErrSupplierQuoteNotFound
	}
	if err != nil {
		return SupplierQuote{}, err
	}
	if version != input.ExpectedVersion {
		return SupplierQuote{}, ErrSupplierQuoteVersion
	}
	if status != "SUBMITTED" || input.Currency != requestCurrency {
		return SupplierQuote{}, ErrInvalidSourcingAction
	}
	_, err = tx.Exec(ctx, `UPDATE supplier_quotes SET quote_reference=$2,amount=$3::numeric,currency=$4,
		delivery_on=$5::date,warranty_months=$6,payment_terms=$7,note=NULLIF($8,''),
		version=version+1,updated_at=now() WHERE id=$1`, quoteID, input.QuoteReference,
		input.Amount, input.Currency, input.DeliveryOn, input.WarrantyMonths, input.PaymentTerms, input.Note)
	if err != nil {
		return SupplierQuote{}, err
	}
	if err = insertResourceAudit(ctx, tx, "supplier_quote", quoteID, "SUPPLIER_QUOTE_UPDATED", user.ID, principal.Roles, status, status, input.CorrelationID); err != nil {
		return SupplierQuote{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return SupplierQuote{}, err
	}
	return s.getSupplierQuote(ctx, quoteID)
}

func (s *Store) SelectSupplierQuote(ctx context.Context, principal auth.Principal, quoteID string, input SelectSupplierQuoteInput) (SourcingCase, error) {
	if !canManageSourcing(principal) {
		return SourcingCase{}, ErrForbidden
	}
	input.Comment = strings.TrimSpace(input.Comment)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if len([]rune(input.Comment)) < 10 || len([]rune(input.Comment)) > 2000 ||
		input.ExpectedCaseVersion < 1 || input.ExpectedQuoteVersion < 1 ||
		!idempotencyPattern.MatchString(input.IdempotencyKey) {
		return SourcingCase{}, &ValidationError{Violations: []FieldViolation{{Field: "comment", Message: "Cần ghi lý do lựa chọn từ 10 đến 2.000 ký tự và phiên bản hợp lệ."}}}
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return SourcingCase{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return SourcingCase{}, err
	}
	var caseID, requestID, caseStatus, quoteStatus, complianceStatus string
	var caseVersion, quoteVersion int64
	err = tx.QueryRow(ctx, `
		SELECT sc.id,sc.purchase_request_id,sc.status,sc.version,q.status,q.version,s.compliance_status
		FROM supplier_quotes q JOIN sourcing_cases sc ON sc.id=q.sourcing_case_id
		JOIN suppliers s ON s.id=q.supplier_id
		WHERE q.id=$1 AND sc.organization_id=$2 FOR UPDATE OF sc,q
	`, quoteID, user.OrganizationID).Scan(&caseID, &requestID, &caseStatus, &caseVersion,
		&quoteStatus, &quoteVersion, &complianceStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return SourcingCase{}, ErrSupplierQuoteNotFound
	}
	if err != nil {
		return SourcingCase{}, err
	}
	if caseVersion != input.ExpectedCaseVersion || quoteVersion != input.ExpectedQuoteVersion {
		return SourcingCase{}, ErrSupplierQuoteVersion
	}
	if caseStatus != "OPEN" || quoteStatus != "SUBMITTED" || complianceStatus == "BLOCKED" {
		return SourcingCase{}, ErrInvalidSourcingAction
	}
	_, err = tx.Exec(ctx, `UPDATE supplier_quotes SET status=CASE WHEN id=$2 THEN 'SELECTED' ELSE 'REJECTED' END,version=version+1,updated_at=now() WHERE sourcing_case_id=$1 AND status='SUBMITTED'`, caseID, quoteID)
	if err != nil {
		return SourcingCase{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE sourcing_cases SET status='AWARDED',selected_quote_id=$2,awarded_by=$3,awarded_at=now(),version=version+1,updated_at=now() WHERE id=$1`, caseID, quoteID, user.ID)
	if err != nil {
		return SourcingCase{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO sourcing_events(sourcing_case_id,quote_id,event_type,actor_id,actor_roles,comment,correlation_id,idempotency_key) VALUES($1,$2,'QUOTE_SELECTED',$3,$4,$5,NULLIF($6,''),$7)`, caseID, quoteID, user.ID, principal.Roles, input.Comment, input.CorrelationID, input.IdempotencyKey)
	if err != nil {
		var databaseError *pgconn.PgError
		if errors.As(err, &databaseError) && databaseError.Code == "23505" {
			return SourcingCase{}, ErrIdempotencyConflict
		}
		return SourcingCase{}, err
	}
	if err = insertResourceAudit(ctx, tx, "sourcing_case", caseID, "SUPPLIER_QUOTE_SELECTED", user.ID, principal.Roles, "OPEN", "AWARDED", input.CorrelationID); err != nil {
		return SourcingCase{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return SourcingCase{}, err
	}
	return s.getSourcingCase(ctx, principal, requestID)
}

func validateSupplierQuote(reference, amount, currency, deliveryOn *string, warrantyMonths int, paymentTerms, note *string) error {
	*reference = strings.ToUpper(strings.TrimSpace(*reference))
	*amount = strings.TrimSpace(*amount)
	*currency = strings.ToUpper(strings.TrimSpace(*currency))
	*deliveryOn = strings.TrimSpace(*deliveryOn)
	*paymentTerms = strings.TrimSpace(*paymentTerms)
	*note = strings.TrimSpace(*note)
	var violations []FieldViolation
	if len(*reference) < 2 || len(*reference) > 100 {
		violations = append(violations, FieldViolation{Field: "quoteReference", Message: "Mã báo giá phải có từ 2 đến 100 ký tự."})
	}
	if number, valid := decimal(*amount, unitPricePattern, true); !valid || number.Cmp(maxMoney) > 0 {
		violations = append(violations, FieldViolation{Field: "amount", Message: "Giá trị báo giá phải là số dương hợp lệ."})
	}
	if !currencyPattern.MatchString(*currency) {
		violations = append(violations, FieldViolation{Field: "currency", Message: "Tiền tệ không hợp lệ."})
	}
	date, err := time.Parse(time.DateOnly, *deliveryOn)
	if err != nil || date.Before(time.Now().Truncate(24*time.Hour)) {
		violations = append(violations, FieldViolation{Field: "deliveryOn", Message: "Ngày giao dự kiến không được trong quá khứ."})
	}
	if warrantyMonths < 0 || warrantyMonths > 240 {
		violations = append(violations, FieldViolation{Field: "warrantyMonths", Message: "Bảo hành phải từ 0 đến 240 tháng."})
	}
	if len([]rune(*paymentTerms)) < 3 || len([]rune(*paymentTerms)) > 500 {
		violations = append(violations, FieldViolation{Field: "paymentTerms", Message: "Điều khoản thanh toán phải có từ 3 đến 500 ký tự."})
	}
	if len([]rune(*note)) > 2000 {
		violations = append(violations, FieldViolation{Field: "note", Message: "Ghi chú không được quá 2.000 ký tự."})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}

func (s *Store) listSupplierQuotes(ctx context.Context, caseID string) ([]SupplierQuote, error) {
	rows, err := s.database.Query(ctx, supplierQuoteSelect+` WHERE q.sourcing_case_id=$1 ORDER BY total_score DESC,q.created_at`, caseID)
	if err != nil {
		return nil, fmt.Errorf("list supplier quotes: %w", err)
	}
	defer rows.Close()
	result := make([]SupplierQuote, 0)
	for rows.Next() {
		item, scanErr := scanSupplierQuote(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

const supplierQuoteSelect = `
	WITH scored AS (
		SELECT q.*,s.code supplier_code,s.name supplier_name,s.performance_score,s.compliance_status,s.risk_level,
		       min(q.amount) OVER(PARTITION BY q.sourcing_case_id) min_amount,
		       min(q.delivery_on) OVER(PARTITION BY q.sourcing_case_id) min_delivery
		FROM supplier_quotes q JOIN suppliers s ON s.id=q.supplier_id
	)
	SELECT q.id,q.sourcing_case_id,q.supplier_id,q.supplier_code,q.supplier_name,q.quote_reference,
	       q.amount::text,q.currency,q.delivery_on::text,q.warranty_months,q.payment_terms,COALESCE(q.note,''),
	       q.status,
	       round((q.min_amount/q.amount*100)::numeric,2)::float8 price_score,
	       greatest(0,100-((q.delivery_on-q.min_delivery)*5))::float8 delivery_score,
	       COALESCE(q.performance_score,60)::float8 quality_score,
	       greatest(0,(CASE q.compliance_status WHEN 'VERIFIED' THEN 100 WHEN 'PENDING' THEN 65 WHEN 'EXPIRED' THEN 30 ELSE 0 END)
	          -(CASE q.risk_level WHEN 'HIGH' THEN 35 WHEN 'MEDIUM' THEN 15 ELSE 0 END))::float8 compliance_score,
	       round(((q.min_amount/q.amount*100)*0.40
	          +greatest(0,100-((q.delivery_on-q.min_delivery)*5))*0.25
	          +COALESCE(q.performance_score,60)*0.20
	          +greatest(0,(CASE q.compliance_status WHEN 'VERIFIED' THEN 100 WHEN 'PENDING' THEN 65 WHEN 'EXPIRED' THEN 30 ELSE 0 END)
	             -(CASE q.risk_level WHEN 'HIGH' THEN 35 WHEN 'MEDIUM' THEN 15 ELSE 0 END))*0.15)::numeric,2)::float8 total_score,
	       q.version
	FROM scored q`

type rowScanner interface{ Scan(...any) error }

func scanSupplierQuote(row rowScanner) (SupplierQuote, error) {
	var item SupplierQuote
	err := row.Scan(&item.ID, &item.SourcingCaseID, &item.SupplierID, &item.SupplierCode,
		&item.SupplierName, &item.QuoteReference, &item.Amount, &item.Currency, &item.DeliveryOn,
		&item.WarrantyMonths, &item.PaymentTerms, &item.Note, &item.Status, &item.PriceScore,
		&item.DeliveryScore, &item.QualityScore, &item.ComplianceScore, &item.TotalScore, &item.Version)
	if err == nil {
		switch {
		case item.TotalScore >= 85:
			item.Recommendation = "Rất phù hợp"
		case item.TotalScore >= 70:
			item.Recommendation = "Phù hợp"
		default:
			item.Recommendation = "Cần rà soát"
		}
	}
	return item, err
}

func (s *Store) getSupplierQuote(ctx context.Context, quoteID string) (SupplierQuote, error) {
	item, err := scanSupplierQuote(s.database.QueryRow(ctx, supplierQuoteSelect+` WHERE q.id=$1`, quoteID))
	if errors.Is(err, pgx.ErrNoRows) {
		return SupplierQuote{}, ErrSupplierQuoteNotFound
	}
	return item, err
}

func (s *Store) getSourcingCase(ctx context.Context, principal auth.Principal, requestID string) (SourcingCase, error) {
	board, err := s.SourcingBoard(ctx, principal)
	if err != nil {
		return SourcingCase{}, err
	}
	for _, item := range board.Items {
		if item.PurchaseRequestID == requestID {
			return item, nil
		}
	}
	return SourcingCase{}, ErrSourcingCaseNotFound
}

func subtractMoney(left, right string) (string, bool) {
	leftNumber, leftOK := new(big.Rat).SetString(left)
	rightNumber, rightOK := new(big.Rat).SetString(right)
	if !leftOK || !rightOK || leftNumber.Cmp(rightNumber) <= 0 {
		return "0", leftOK && rightOK
	}
	return new(big.Rat).Sub(leftNumber, rightNumber).FloatString(4), true
}
