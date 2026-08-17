package httpapi

import (
	"net/http"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/dx-os-lab/dx-os/backend/internal/procurement"
	"github.com/go-chi/chi/v5"
)

type auditCaseBody struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	Severity        string `json:"severity"`
	ResourceType    string `json:"resourceType"`
	ResourceID      string `json:"resourceId"`
	OwnerUserID     string `json:"ownerUserId"`
	DueOn           string `json:"dueOn"`
	Status          string `json:"status"`
	Resolution      string `json:"resolution"`
	ExpectedVersion int64  `json:"expectedVersion"`
}

func (body auditCaseBody) input(r *http.Request) procurement.AuditCaseInput {
	return procurement.AuditCaseInput{
		Title: body.Title, Description: body.Description, Severity: body.Severity,
		ResourceType: body.ResourceType, ResourceID: body.ResourceID,
		OwnerUserID: body.OwnerUserID, DueOn: body.DueOn, Status: body.Status,
		Resolution: body.Resolution, ExpectedVersion: body.ExpectedVersion,
		CorrelationID: correlationIDFromContext(r.Context()),
	}
}

func (a *api) listAuditCases(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	result, err := a.enterprise.ListAuditCases(r.Context(), principal)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) createAuditCase(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	var body auditCaseBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.enterprise.CreateAuditCase(r.Context(), principal, body.input(r))
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (a *api) updateAuditCase(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	caseID := chi.URLParam(r, "caseID")
	if !uuidPattern.MatchString(caseID) {
		writeValidationProblem(w, r, "invalid-audit-case-id", "The audit case ID must be a valid UUID.", nil)
		return
	}
	var body auditCaseBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.enterprise.UpdateAuditCase(r.Context(), principal, caseID, body.input(r))
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) downloadEvidencePackage(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	requestID := chi.URLParam(r, "requestID")
	if !uuidPattern.MatchString(requestID) {
		writeValidationProblem(w, r, "invalid-purchase-request-id", "The purchase request ID must be a valid UUID.", nil)
		return
	}
	result, err := a.enterprise.EvidencePackage(r.Context(), principal, requestID)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="dx-os-evidence-`+result.Request.RequestCode+`.json"`)
	writeJSON(w, http.StatusOK, result)
}

func (a *api) enterprisePrincipal(w http.ResponseWriter, r *http.Request) (auth.Principal, bool) {
	principal, ok := principalFromContext(r.Context())
	if !ok || a.enterprise == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Enterprise procurement service is unavailable.")
		return auth.Principal{}, false
	}
	return principal, true
}
