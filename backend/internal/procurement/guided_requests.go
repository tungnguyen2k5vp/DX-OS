package procurement

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
)

type CatalogItem struct {
	ID                 string `json:"id"`
	Code               string `json:"code"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	Category           string `json:"category"`
	Unit               string `json:"unit"`
	ReferenceUnitPrice string `json:"referenceUnitPrice"`
	Currency           string `json:"currency"`
}

type Catalog struct {
	Items []CatalogItem `json:"items"`
	Total int           `json:"total"`
}

type DuplicateCheckInput struct {
	Title            string
	CostCenter       string
	TotalAmount      string
	ExcludeRequestID string
}

type DuplicateCandidate struct {
	PurchaseRequestID string  `json:"purchaseRequestId"`
	RequestCode       string  `json:"requestCode"`
	Title             string  `json:"title"`
	Status            Status  `json:"status"`
	TotalAmount       string  `json:"totalAmount"`
	Currency          string  `json:"currency"`
	Similarity        float64 `json:"similarity"`
	Reason            string  `json:"reason"`
}

type DuplicateCheckResult struct {
	PotentialDuplicate bool                 `json:"potentialDuplicate"`
	Items              []DuplicateCandidate `json:"items"`
}

func (s *Store) ListCatalog(ctx context.Context, principal auth.Principal) (Catalog, error) {
	if _, err := ScopeFor(principal); err != nil {
		return Catalog{}, err
	}
	user, err := s.ensureUser(ctx, principal)
	if err != nil {
		return Catalog{}, err
	}
	rows, err := s.database.Query(ctx, `
		SELECT id,code,name,description,category,unit,reference_unit_price::text,currency
		FROM procurement_catalog_items
		WHERE organization_id=$1 AND active
		ORDER BY category,name
	`, user.OrganizationID)
	if err != nil {
		return Catalog{}, fmt.Errorf("list procurement catalog: %w", err)
	}
	defer rows.Close()
	result := Catalog{Items: make([]CatalogItem, 0)}
	for rows.Next() {
		var item CatalogItem
		if err = rows.Scan(&item.ID, &item.Code, &item.Name, &item.Description, &item.Category,
			&item.Unit, &item.ReferenceUnitPrice, &item.Currency); err != nil {
			return Catalog{}, fmt.Errorf("scan procurement catalog: %w", err)
		}
		result.Items = append(result.Items, item)
	}
	result.Total = len(result.Items)
	return result, rows.Err()
}

func (s *Store) CheckDuplicateRequests(
	ctx context.Context,
	principal auth.Principal,
	input DuplicateCheckInput,
) (DuplicateCheckResult, error) {
	if _, err := ScopeFor(principal); err != nil {
		return DuplicateCheckResult{}, err
	}
	input.Title = strings.TrimSpace(input.Title)
	input.CostCenter = strings.TrimSpace(input.CostCenter)
	input.TotalAmount = strings.TrimSpace(input.TotalAmount)
	if len([]rune(input.Title)) < 3 || len([]rune(input.Title)) > 255 {
		return DuplicateCheckResult{}, &ValidationError{Violations: []FieldViolation{{
			Field: "title", Message: "Tiêu đề phải có từ 3 đến 255 ký tự.",
		}}}
	}
	if input.TotalAmount != "" {
		if _, valid := decimal(input.TotalAmount, unitPricePattern, false); !valid {
			return DuplicateCheckResult{}, &ValidationError{Violations: []FieldViolation{{
				Field: "totalAmount", Message: "Tổng tiền phải là số không âm hợp lệ.",
			}}}
		}
	}
	user, err := s.ensureUser(ctx, principal)
	if err != nil {
		return DuplicateCheckResult{}, err
	}
	rows, err := s.database.Query(ctx, `
		SELECT pr.id,pr.request_code,pr.title,pr.status,pr.total_amount::text,pr.currency,pr.cost_center
		FROM purchase_requests pr
		JOIN departments d ON d.id=pr.department_id
		WHERE d.organization_id=$1
		  AND pr.department_id=$2
		  AND pr.status NOT IN ('REJECTED','CANCELLED')
		  AND pr.created_at >= now()-interval '90 days'
		  AND ($3='' OR pr.id::text<>$3)
		ORDER BY pr.created_at DESC
		LIMIT 100
	`, user.OrganizationID, user.DepartmentID, input.ExcludeRequestID)
	if err != nil {
		return DuplicateCheckResult{}, fmt.Errorf("find duplicate requests: %w", err)
	}
	defer rows.Close()
	titleTokens := normalizedTokens(input.Title)
	result := DuplicateCheckResult{Items: make([]DuplicateCandidate, 0)}
	for rows.Next() {
		var item DuplicateCandidate
		var costCenter string
		if err = rows.Scan(&item.PurchaseRequestID, &item.RequestCode, &item.Title, &item.Status,
			&item.TotalAmount, &item.Currency, &costCenter); err != nil {
			return DuplicateCheckResult{}, err
		}
		item.Similarity = tokenSimilarity(titleTokens, normalizedTokens(item.Title))
		amountClose := input.TotalAmount != "" && moneyWithinPercent(input.TotalAmount, item.TotalAmount, 10)
		costCenterSame := input.CostCenter != "" && strings.EqualFold(input.CostCenter, costCenter)
		if item.Similarity < 0.6 && !(item.Similarity >= 0.45 && amountClose && costCenterSame) {
			continue
		}
		reasons := []string{fmt.Sprintf("Tiêu đề giống %.0f%%", item.Similarity*100)}
		if amountClose {
			reasons = append(reasons, "giá trị gần nhau")
		}
		if costCenterSame {
			reasons = append(reasons, "cùng trung tâm chi phí")
		}
		item.Reason = strings.Join(reasons, ", ")
		result.Items = append(result.Items, item)
	}
	sort.Slice(result.Items, func(i, j int) bool {
		return result.Items[i].Similarity > result.Items[j].Similarity
	})
	if len(result.Items) > 5 {
		result.Items = result.Items[:5]
	}
	result.PotentialDuplicate = len(result.Items) > 0
	return result, rows.Err()
}

func normalizedTokens(value string) map[string]struct{} {
	normalized := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, value)
	result := make(map[string]struct{})
	for _, token := range strings.Fields(normalized) {
		if len([]rune(token)) >= 2 {
			result[token] = struct{}{}
		}
	}
	return result
}

func tokenSimilarity(left, right map[string]struct{}) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	intersection := 0
	union := make(map[string]struct{}, len(left)+len(right))
	for token := range left {
		union[token] = struct{}{}
		if _, ok := right[token]; ok {
			intersection++
		}
	}
	for token := range right {
		union[token] = struct{}{}
	}
	return float64(intersection) / float64(len(union))
}

func moneyWithinPercent(left, right string, percent float64) bool {
	leftNumber, leftOK := newFloat(left)
	rightNumber, rightOK := newFloat(right)
	if !leftOK || !rightOK || leftNumber == 0 || rightNumber == 0 {
		return false
	}
	return math.Abs(leftNumber-rightNumber)/math.Max(leftNumber, rightNumber)*100 <= percent
}

func newFloat(value string) (float64, bool) {
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, false
	}
	return number, true
}
