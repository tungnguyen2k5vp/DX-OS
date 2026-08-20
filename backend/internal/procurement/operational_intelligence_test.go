package procurement

import (
	"testing"
	"time"
)

func TestTokenSimilarityRecognizesRelatedVietnameseTitles(t *testing.T) {
	left := normalizedTokens("Mua laptop phục vụ nhóm thiết kế")
	right := normalizedTokens("Mua laptop cho nhóm thiết kế báo cáo")
	if similarity := tokenSimilarity(left, right); similarity < 0.45 {
		t.Fatalf("expected related titles to have useful similarity, got %f", similarity)
	}
}

func TestMoneyWithinPercentUsesConfiguredTolerance(t *testing.T) {
	if !moneyWithinPercent("50000000", "52500000", 10) {
		t.Fatal("expected amounts within five percent to match")
	}
	if moneyWithinPercent("50000000", "70000000", 10) {
		t.Fatal("expected materially different amounts not to match")
	}
}

func TestValidateApprovalRuleRequiresAtLeastOneApprovalLevel(t *testing.T) {
	input := ApprovalRuleInput{
		Name: "Quy tắc thử nghiệm", Currency: "VND", MinimumAmount: "0",
		RequiresManager: false, RequiresFinance: false, Priority: 10,
	}
	if err := validateApprovalRule(&input, false); err == nil {
		t.Fatal("expected a rule without approval levels to be rejected")
	}
}

func TestValidateSupplierQuoteAcceptsFutureDelivery(t *testing.T) {
	reference := "BG-TEST"
	amount := "25000000"
	currency := "vnd"
	delivery := time.Now().AddDate(0, 0, 7).Format(time.DateOnly)
	paymentTerms := "Thanh toán sau nghiệm thu"
	note := ""
	if err := validateSupplierQuote(&reference, &amount, &currency, &delivery, 12, &paymentTerms, &note); err != nil {
		t.Fatalf("expected valid quote, got %v", err)
	}
	if currency != "VND" {
		t.Fatalf("expected currency normalization, got %s", currency)
	}
}

func TestRoleConflictLabelsDetectsSegregationOfDuties(t *testing.T) {
	labels := roleConflictLabels([]string{"finance", "auditor"})
	if len(labels) != 1 {
		t.Fatalf("expected one finance/auditor conflict, got %#v", labels)
	}
}
