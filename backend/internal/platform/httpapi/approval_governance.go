package httpapi

import (
	"net/http"

	"github.com/dx-os-lab/dx-os/backend/internal/procurement"
	"github.com/go-chi/chi/v5"
)

type delegationBody struct {
	DelegateUserID string `json:"delegateUserId"`
	StartsOn       string `json:"startsOn"`
	EndsOn         string `json:"endsOn"`
	Reason         string `json:"reason"`
}

type delegationStatusBody struct {
	Active          bool  `json:"active"`
	ExpectedVersion int64 `json:"expectedVersion"`
}

type approvalRuleBody struct {
	DepartmentID    string `json:"departmentId"`
	Name            string `json:"name"`
	Currency        string `json:"currency"`
	MinimumAmount   string `json:"minimumAmount"`
	MaximumAmount   string `json:"maximumAmount"`
	RequiresManager bool   `json:"requiresManager"`
	RequiresFinance bool   `json:"requiresFinance"`
	Priority        int    `json:"priority"`
	Active          bool   `json:"active"`
	ExpectedVersion int64  `json:"expectedVersion"`
}

func (a *api) getApprovalGovernance(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	result, err := a.enterprise.ApprovalGovernance(r.Context(), principal)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) createApprovalDelegation(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	var body delegationBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.enterprise.CreateApprovalDelegation(r.Context(), principal, procurement.CreateDelegationInput{
		DelegateUserID: body.DelegateUserID, StartsOn: body.StartsOn, EndsOn: body.EndsOn,
		Reason: body.Reason, CorrelationID: correlationIDFromContext(r.Context()),
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (a *api) setApprovalDelegationActive(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "delegationID")
	if !uuidPattern.MatchString(id) {
		writeValidationProblem(w, r, "invalid-delegation-id", "Mã ủy quyền không hợp lệ.", nil)
		return
	}
	var body delegationStatusBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.enterprise.SetApprovalDelegationActive(r.Context(), principal, id, procurement.SetDelegationActiveInput{
		Active: body.Active, ExpectedVersion: body.ExpectedVersion, CorrelationID: correlationIDFromContext(r.Context()),
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) createApprovalRule(w http.ResponseWriter, r *http.Request) {
	a.saveApprovalRule(w, r, "")
}

func (a *api) updateApprovalRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "ruleID")
	if !uuidPattern.MatchString(id) {
		writeValidationProblem(w, r, "invalid-approval-rule-id", "Mã quy tắc không hợp lệ.", nil)
		return
	}
	a.saveApprovalRule(w, r, id)
}

func (a *api) saveApprovalRule(w http.ResponseWriter, r *http.Request, id string) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	var body approvalRuleBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.enterprise.SaveApprovalRule(r.Context(), principal, id, procurement.ApprovalRuleInput{
		DepartmentID: body.DepartmentID, Name: body.Name, Currency: body.Currency,
		MinimumAmount: body.MinimumAmount, MaximumAmount: body.MaximumAmount,
		RequiresManager: body.RequiresManager, RequiresFinance: body.RequiresFinance,
		Priority: body.Priority, Active: body.Active, ExpectedVersion: body.ExpectedVersion,
		CorrelationID: correlationIDFromContext(r.Context()),
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	status := http.StatusOK
	if id == "" {
		status = http.StatusCreated
	}
	writeJSON(w, status, result)
}
