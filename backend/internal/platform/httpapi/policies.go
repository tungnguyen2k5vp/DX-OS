package httpapi

import (
	"net/http"
	"strings"

	"github.com/dx-os-lab/dx-os/backend/internal/procurement"
	"github.com/go-chi/chi/v5"
)

type slaPolicyBody struct {
	TargetHours     int   `json:"targetHours"`
	Active          bool  `json:"active"`
	ExpectedVersion int64 `json:"expectedVersion"`
}

type attachmentPolicyBody struct {
	ThresholdAmount      string `json:"thresholdAmount"`
	RequiredDocumentType string `json:"requiredDocumentType"`
	Active               bool   `json:"active"`
	ExpectedVersion      int64  `json:"expectedVersion"`
}

func (a *api) getPolicyCenter(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromContext(r.Context())
	if !ok || a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return
	}
	result, err := a.procurement.PolicyCenter(r.Context(), principal)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) updateSLAPolicy(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromContext(r.Context())
	if !ok || a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return
	}
	processName := strings.TrimSpace(chi.URLParam(r, "processName"))
	if processName == "" || len(processName) > 80 {
		writeValidationProblem(w, r, "invalid-policy-id", "The process name is invalid.", nil)
		return
	}
	var body slaPolicyBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.procurement.UpdateSLAPolicy(r.Context(), principal, processName, procurement.UpdateSLAPolicyInput{
		TargetHours: body.TargetHours, Active: body.Active, ExpectedVersion: body.ExpectedVersion,
		CorrelationID: correlationIDFromContext(r.Context()),
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) updateAttachmentPolicy(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromContext(r.Context())
	if !ok || a.procurement == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "Procurement service is unavailable.")
		return
	}
	ruleID := strings.TrimSpace(chi.URLParam(r, "ruleID"))
	if !uuidPattern.MatchString(ruleID) {
		writeValidationProblem(w, r, "invalid-policy-id", "The attachment policy ID must be a valid UUID.", nil)
		return
	}
	var body attachmentPolicyBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.procurement.UpdateAttachmentPolicy(r.Context(), principal, ruleID, procurement.UpdateAttachmentPolicyInput{
		ThresholdAmount: body.ThresholdAmount, RequiredDocumentType: body.RequiredDocumentType,
		Active: body.Active, ExpectedVersion: body.ExpectedVersion,
		CorrelationID: correlationIDFromContext(r.Context()),
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
