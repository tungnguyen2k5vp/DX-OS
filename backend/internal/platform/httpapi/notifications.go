package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/dx-os-lab/dx-os/backend/internal/notifications"
	"github.com/go-chi/chi/v5"
)

func (a *api) listNotifications(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Cần access token hợp lệ để tiếp tục.")
		return
	}
	if a.notifications == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Dịch vụ chưa sẵn sàng", "Dịch vụ thông báo hiện không sẵn sàng.")
		return
	}
	input := notifications.ListInput{Page: 1, PageSize: 20}
	var invalid bool
	if value := strings.TrimSpace(r.URL.Query().Get("page")); value != "" {
		input.Page, _ = strconv.Atoi(value)
		invalid = input.Page < 1
	}
	if value := strings.TrimSpace(r.URL.Query().Get("pageSize")); value != "" {
		input.PageSize, _ = strconv.Atoi(value)
		invalid = invalid || input.PageSize < 1 || input.PageSize > 50
	}
	if value := strings.TrimSpace(r.URL.Query().Get("unreadOnly")); value != "" {
		parsed, err := strconv.ParseBool(value)
		invalid = invalid || err != nil
		input.UnreadOnly = parsed
	}
	for key := range r.URL.Query() {
		if key != "page" && key != "pageSize" && key != "unreadOnly" {
			invalid = true
		}
	}
	if invalid {
		writeValidationProblem(w, r, "invalid-notification-query", "Tham số truy vấn thông báo không hợp lệ.", nil)
		return
	}
	result, err := a.notifications.List(r.Context(), principal, input)
	if err != nil {
		a.writeNotificationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) markNotificationRead(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Cần access token hợp lệ để tiếp tục.")
		return
	}
	if a.notifications == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Dịch vụ chưa sẵn sàng", "Dịch vụ thông báo hiện không sẵn sàng.")
		return
	}
	notificationID := strings.TrimSpace(chi.URLParam(r, "notificationID"))
	if !uuidPattern.MatchString(notificationID) {
		writeValidationProblem(w, r, "invalid-notification-id", "Mã thông báo phải là UUID hợp lệ.", nil)
		return
	}
	if err := a.notifications.MarkRead(r.Context(), principal, notificationID); err != nil {
		a.writeNotificationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"read": true})
}

func (a *api) markAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Cần access token hợp lệ để tiếp tục.")
		return
	}
	if a.notifications == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "service-unavailable", "Dịch vụ chưa sẵn sàng", "Dịch vụ thông báo hiện không sẵn sàng.")
		return
	}
	count, err := a.notifications.MarkAllRead(r.Context(), principal)
	if err != nil {
		a.writeNotificationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"markedRead": count})
}

func (a *api) writeNotificationError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, notifications.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "notification-forbidden", "Không có quyền truy cập", "Tài khoản không có quyền xem thông báo.")
	case errors.Is(err, notifications.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "notification-not-found", "Không tìm thấy", "Không tìm thấy thông báo.")
	default:
		a.logger.Error("notification request failed", "error", err, "correlation_id", correlationIDFromContext(r.Context()))
		writeProblem(w, r, http.StatusInternalServerError, "internal", "Lỗi máy chủ", "Không thể hoàn tất thao tác thông báo.")
	}
}
