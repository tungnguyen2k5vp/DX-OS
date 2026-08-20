package httpapi

import (
	"net/http"

	"github.com/dx-os-lab/dx-os/backend/internal/procurement"
)

type duplicateRequestBody struct {
	Title            string `json:"title"`
	CostCenter       string `json:"costCenter"`
	TotalAmount      string `json:"totalAmount"`
	ExcludeRequestID string `json:"excludeRequestId"`
}

func (a *api) listProcurementCatalog(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	result, err := a.enterprise.ListCatalog(r.Context(), principal)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) checkDuplicatePurchaseRequests(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	var body duplicateRequestBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.enterprise.CheckDuplicateRequests(r.Context(), principal, procurement.DuplicateCheckInput{
		Title: body.Title, CostCenter: body.CostCenter, TotalAmount: body.TotalAmount,
		ExcludeRequestID: body.ExcludeRequestID,
	})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
