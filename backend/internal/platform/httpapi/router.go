package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/dx-os-lab/dx-os/backend/internal/notifications"
	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/dx-os-lab/dx-os/backend/internal/procurement"
	"github.com/dx-os-lab/dx-os/backend/internal/reporting"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type tokenVerifier interface {
	Verify(context.Context, string) (auth.Principal, error)
}

type purchaseRequestService interface {
	Create(context.Context, auth.Principal, procurement.CreateInput, string) (procurement.PurchaseRequest, error)
	List(context.Context, auth.Principal, procurement.ListInput) (procurement.ListResult, error)
	Get(context.Context, auth.Principal, string) (procurement.PurchaseRequest, error)
	Timeline(context.Context, auth.Principal, string, procurement.TimelineInput) (procurement.TimelineResult, error)
	ListComments(context.Context, auth.Principal, string) (procurement.CommentList, error)
	AddComment(context.Context, auth.Principal, string, procurement.CommentInput) (procurement.Comment, error)
	TaskSummary(context.Context, auth.Principal) (procurement.WorkSummary, error)
	ListSuppliers(context.Context, auth.Principal) (procurement.SupplierList, error)
	CreateSupplier(context.Context, auth.Principal, procurement.SupplierInput, string) (procurement.Supplier, error)
	UpdateSupplier(context.Context, auth.Principal, string, procurement.UpdateSupplierInput, string) (procurement.Supplier, error)
	OperationsBoard(context.Context, auth.Principal) (procurement.OperationsBoard, error)
	CreatePurchaseOrder(context.Context, auth.Principal, procurement.CreatePurchaseOrderInput) (procurement.PurchaseOrder, error)
	ConfirmReceipt(context.Context, auth.Principal, string, procurement.ConfirmReceiptInput) (procurement.PurchaseOrder, error)
	InvoiceBoard(context.Context, auth.Principal) (procurement.InvoiceBoard, error)
	CreateInvoice(context.Context, auth.Principal, procurement.InvoiceInput) (procurement.InvoiceBoardItem, error)
	UpdateInvoice(context.Context, auth.Principal, string, procurement.UpdateInvoiceInput) (procurement.InvoiceBoardItem, error)
	TransitionInvoice(context.Context, auth.Principal, string, procurement.InvoiceActionInput) (procurement.InvoiceBoardItem, error)
	PolicyCenter(context.Context, auth.Principal) (procurement.PolicyCenter, error)
	UpdateSLAPolicy(context.Context, auth.Principal, string, procurement.UpdateSLAPolicyInput) (procurement.SLAPolicy, error)
	UpdateAttachmentPolicy(context.Context, auth.Principal, string, procurement.UpdateAttachmentPolicyInput) (procurement.AttachmentPolicy, error)
	GetBudgetSummary(context.Context, auth.Principal, procurement.BudgetSummaryInput) (procurement.BudgetSummary, error)
	BudgetCheck(context.Context, auth.Principal, string) (procurement.BudgetCheck, error)
	BudgetDashboard(context.Context, auth.Principal) (procurement.BudgetDashboard, error)
	AdjustBudget(context.Context, auth.Principal, string, procurement.AdjustBudgetInput) (procurement.BudgetAllocation, error)
	Update(context.Context, auth.Principal, string, procurement.UpdateInput, string) (procurement.PurchaseRequest, error)
	Transition(context.Context, auth.Principal, string, procurement.TransitionInput) (procurement.PurchaseRequest, error)
	UploadAttachment(context.Context, auth.Principal, string, procurement.UploadAttachmentInput) (procurement.Attachment, error)
	ListAttachments(context.Context, auth.Principal, string) (procurement.AttachmentList, error)
	DownloadAttachment(context.Context, auth.Principal, string, string) (procurement.AttachmentContent, error)
	DeleteAttachment(context.Context, auth.Principal, string, string, string) error
}

type reportingService interface {
	Dashboard(context.Context, auth.Principal, reporting.DashboardInput) (reporting.Dashboard, error)
	AuditCenter(context.Context, auth.Principal, reporting.AuditInput) (reporting.AuditCenter, error)
}

type notificationService interface {
	List(context.Context, auth.Principal, notifications.ListInput) (notifications.ListResult, error)
	MarkRead(context.Context, auth.Principal, string) error
	MarkAllRead(context.Context, auth.Principal) (int64, error)
}

type Dependencies struct {
	AllowedOrigin string
	Database      *pgxpool.Pool
	Logger        *slog.Logger
	Notifications notificationService
	Procurement   purchaseRequestService
	Reporting     reportingService
	TokenVerifier tokenVerifier
	RateLimit     int
	RateWindow    time.Duration
}

type api struct {
	allowedOrigin string
	database      *pgxpool.Pool
	logger        *slog.Logger
	notifications notificationService
	procurement   purchaseRequestService
	reporting     reportingService
	tokenVerifier tokenVerifier
	rateLimiter   *principalRateLimiter
}

func New(deps Dependencies) http.Handler {
	rateLimit := deps.RateLimit
	if rateLimit <= 0 {
		rateLimit = 120
	}
	rateWindow := deps.RateWindow
	if rateWindow <= 0 {
		rateWindow = time.Minute
	}
	server := &api{
		allowedOrigin: deps.AllowedOrigin,
		database:      deps.Database,
		logger:        deps.Logger,
		notifications: deps.Notifications,
		procurement:   deps.Procurement,
		reporting:     deps.Reporting,
		tokenVerifier: deps.TokenVerifier,
		rateLimiter:   newPrincipalRateLimiter(rateLimit, rateWindow),
	}

	router := chi.NewRouter()
	router.Use(server.recoverer)
	router.Use(server.correlationID)
	router.Use(server.accessLog)
	router.Use(server.securityHeaders)
	router.Use(server.cors)

	router.Get("/health/live", server.live)
	router.Get("/health/ready", server.ready)
	router.Route("/api/v1", func(r chi.Router) {
		r.Use(server.authenticate)
		r.Use(server.principalRateLimit)
		r.Get("/me", server.me)
		r.Get("/me/tasks-summary", server.getTaskSummary)
		r.Get("/me/notifications", server.listNotifications)
		r.Post("/me/notifications/read-all", server.markAllNotificationsRead)
		r.Post("/me/notifications/{notificationID}/read", server.markNotificationRead)
		r.Post("/purchase-requests", server.createPurchaseRequest)
		r.Get("/purchase-requests", server.listPurchaseRequests)
		r.Get("/purchase-requests/{requestID}", server.getPurchaseRequest)
		r.Get("/purchase-requests/{requestID}/budget-check", server.getPurchaseRequestBudgetCheck)
		r.Get("/purchase-requests/{requestID}/timeline", server.getPurchaseRequestTimeline)
		r.Get("/purchase-requests/{requestID}/comments", server.listPurchaseRequestComments)
		r.Post("/purchase-requests/{requestID}/comments", server.addPurchaseRequestComment)
		r.Patch("/purchase-requests/{requestID}", server.updatePurchaseRequest)
		r.Post("/purchase-requests/{requestID}/transitions", server.transitionPurchaseRequest)
		r.Get("/purchase-requests/{requestID}/attachments", server.listPurchaseRequestAttachments)
		r.Post("/purchase-requests/{requestID}/attachments", server.uploadPurchaseRequestAttachment)
		r.Get("/purchase-requests/{requestID}/attachments/{attachmentID}/content", server.downloadPurchaseRequestAttachment)
		r.Delete("/purchase-requests/{requestID}/attachments/{attachmentID}", server.deletePurchaseRequestAttachment)
		r.Get("/budgets/summary", server.getBudgetSummary)
		r.Get("/budgets/dashboard", server.getBudgetDashboard)
		r.Patch("/budgets/allocations/{allocationID}", server.adjustBudgetAllocation)
		r.Get("/reports/procurement", server.getProcurementReport)
		r.Get("/suppliers", server.listSuppliers)
		r.Post("/suppliers", server.createSupplier)
		r.Patch("/suppliers/{supplierID}", server.updateSupplier)
		r.Get("/procurement-operations", server.getOperationsBoard)
		r.Post("/procurement-operations/orders", server.createPurchaseOrder)
		r.Post("/procurement-operations/orders/{requestID}/receipt", server.confirmPurchaseOrderReceipt)
		r.Get("/invoices", server.getInvoiceBoard)
		r.Post("/invoices", server.createInvoice)
		r.Patch("/invoices/{invoiceID}", server.updateInvoice)
		r.Post("/invoices/{invoiceID}/transitions", server.transitionInvoice)
		r.Get("/admin/policies", server.getPolicyCenter)
		r.Patch("/admin/policies/sla/{processName}", server.updateSLAPolicy)
		r.Patch("/admin/policies/attachments/{ruleID}", server.updateAttachmentPolicy)
		r.Get("/audit/events", server.getAuditEvents)
	})

	return router
}

func (a *api) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *api) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.database.Ping(ctx); err != nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "dependency-unavailable", "Service unavailable", "Database readiness check failed.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (a *api) me(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required", "A valid access token is required.")
		return
	}
	writeJSON(w, http.StatusOK, principal)
}

func (a *api) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") || len(header) <= len("Bearer ") {
			w.Header().Set("WWW-Authenticate", `Bearer realm="dx-os"`)
			writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required", "A bearer access token is required.")
			return
		}

		principal, err := a.tokenVerifier.Verify(r.Context(), strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
			writeProblem(w, r, http.StatusUnauthorized, "invalid-token", "Invalid access token", "The access token is invalid or expired.")
			return
		}
		next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), principal)))
	})
}

type contextKey string

const (
	correlationIDKey contextKey = "correlation-id"
	principalKey     contextKey = "principal"
)

func withPrincipal(ctx context.Context, principal auth.Principal) context.Context {
	return context.WithValue(ctx, principalKey, principal)
}

func principalFromContext(ctx context.Context) (auth.Principal, bool) {
	principal, ok := ctx.Value(principalKey).(auth.Principal)
	return principal, ok
}

func (a *api) correlationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Correlation-ID"))
		if id == "" || len(id) > 128 {
			id = randomID()
		}
		w.Header().Set("X-Correlation-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), correlationIDKey, id)))
	})
}

func (a *api) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func (a *api) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && origin == a.allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Correlation-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			if origin != a.allowedOrigin {
				writeProblem(w, r, http.StatusForbidden, "cors-denied", "Origin denied", "The request origin is not allowed.")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *api) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		next.ServeHTTP(w, r)
		a.logger.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"correlation_id", correlationIDFromContext(r.Context()),
		)
	})
}

func (a *api) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				a.logger.Error("panic recovered", "panic", recovered, "stack", string(debug.Stack()))
				writeProblem(w, r, http.StatusInternalServerError, "internal", "Internal server error", "An unexpected error occurred.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type problem struct {
	Type          string `json:"type"`
	Title         string `json:"title"`
	Status        int    `json:"status"`
	Detail        string `json:"detail"`
	Instance      string `json:"instance"`
	Code          string `json:"code"`
	CorrelationID string `json:"correlationId"`
	Errors        any    `json:"errors,omitempty"`
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, code, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem{
		Type:          "https://docs.dx-os.local/problems/" + code,
		Title:         title,
		Status:        status,
		Detail:        detail,
		Instance:      r.URL.Path,
		Code:          code,
		CorrelationID: correlationIDFromContext(r.Context()),
	})
}

func writeValidationProblem(
	w http.ResponseWriter,
	r *http.Request,
	code string,
	detail string,
	errors any,
) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(w).Encode(problem{
		Type:          "https://docs.dx-os.local/problems/" + code,
		Title:         "Validation failed",
		Status:        http.StatusUnprocessableEntity,
		Detail:        detail,
		Instance:      r.URL.Path,
		Code:          code,
		CorrelationID: correlationIDFromContext(r.Context()),
		Errors:        errors,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func correlationIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(correlationIDKey).(string)
	return id
}

func randomID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "correlation-id-unavailable"
	}
	return hex.EncodeToString(bytes)
}
