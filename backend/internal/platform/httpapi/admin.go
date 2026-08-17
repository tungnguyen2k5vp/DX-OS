package httpapi

import (
	"net/http"

	"github.com/dx-os-lab/dx-os/backend/internal/procurement"
	"github.com/go-chi/chi/v5"
)

type adminUserBody struct {
	DisplayName     string `json:"displayName"`
	Email           string `json:"email"`
	DepartmentID    string `json:"departmentId"`
	Active          bool   `json:"active"`
	ExpectedVersion int64  `json:"expectedVersion"`
}

type adminDepartmentBody struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	CostCenter      string `json:"costCenter"`
	ParentID        string `json:"parentId"`
	Active          bool   `json:"active"`
	ExpectedVersion int64  `json:"expectedVersion"`
}

func (a *api) getAdminCenter(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	result, err := a.enterprise.AdminCenter(r.Context(), principal)
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) updateAdminUser(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	userID := chi.URLParam(r, "userID")
	if !uuidPattern.MatchString(userID) {
		writeValidationProblem(w, r, "invalid-user-id", "The user ID must be a valid UUID.", nil)
		return
	}
	var body adminUserBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.enterprise.UpdateAdminUser(r.Context(), principal, userID, procurement.UpdateAdminUserInput{DisplayName: body.DisplayName, Email: body.Email, DepartmentID: body.DepartmentID, Active: body.Active, ExpectedVersion: body.ExpectedVersion, CorrelationID: correlationIDFromContext(r.Context())})
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) createAdminDepartment(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	var body adminDepartmentBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.enterprise.CreateDepartment(r.Context(), principal, departmentInput(body, r))
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (a *api) updateAdminDepartment(w http.ResponseWriter, r *http.Request) {
	principal, ok := a.enterprisePrincipal(w, r)
	if !ok {
		return
	}
	departmentID := chi.URLParam(r, "departmentID")
	if !uuidPattern.MatchString(departmentID) {
		writeValidationProblem(w, r, "invalid-department-id", "The department ID must be a valid UUID.", nil)
		return
	}
	var body adminDepartmentBody
	if err := decodeJSONBody(w, r, &body); err != nil {
		writeRequestBodyProblem(w, r, err)
		return
	}
	result, err := a.enterprise.UpdateDepartment(r.Context(), principal, departmentID, departmentInput(body, r))
	if err != nil {
		a.writeProcurementError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func departmentInput(body adminDepartmentBody, r *http.Request) procurement.SaveDepartmentInput {
	return procurement.SaveDepartmentInput{Code: body.Code, Name: body.Name, CostCenter: body.CostCenter, ParentID: body.ParentID, Active: body.Active, ExpectedVersion: body.ExpectedVersion, CorrelationID: correlationIDFromContext(r.Context())}
}
