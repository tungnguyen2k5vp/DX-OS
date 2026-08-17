package httpapi

import (
	"net/http"

	"github.com/dx-os-lab/dx-os/backend/internal/procurement"
	"github.com/go-chi/chi/v5"
)

type aiDecisionBody struct {
	Status          string `json:"status"`
	Comment         string `json:"comment"`
	ExpectedVersion int64  `json:"expectedVersion"`
}

func (a *api) listAIRecommendations(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	result, err := a.enterprise.ListAIRecommendations(r.Context(), principal)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) generateAIRecommendations(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	result, err := a.enterprise.GenerateAIRecommendations(r.Context(), principal)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) decideAIRecommendation(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "recommendationID")
	if !uuidPattern.MatchString(id) {
		writeValidationProblem(w, r, "invalid-ai-recommendation-id", "The recommendation ID must be a valid UUID.", nil)
		return
	}
	var body aiDecisionBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.enterprise.DecideAIRecommendation(r.Context(), principal, id, procurement.DecideAIRecommendationInput{
		Status: body.Status, Comment: body.Comment, ExpectedVersion: body.ExpectedVersion,
		CorrelationID: correlationIDFromContext(r.Context()),
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
