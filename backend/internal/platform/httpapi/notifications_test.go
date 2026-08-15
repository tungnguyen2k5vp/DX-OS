package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dx-os-lab/dx-os/backend/internal/notifications"
	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
)

type notificationStub struct {
	listInput notifications.ListInput
	markedID  string
	markAll   bool
}

func (s *notificationStub) List(
	_ context.Context,
	_ auth.Principal,
	input notifications.ListInput,
) (notifications.ListResult, error) {
	s.listInput = input
	return notifications.ListResult{
		Items: []notifications.Notification{}, Page: input.Page, PageSize: input.PageSize,
		UnreadCount: 2,
	}, nil
}

func (s *notificationStub) MarkRead(
	_ context.Context,
	_ auth.Principal,
	notificationID string,
) error {
	s.markedID = notificationID
	return nil
}

func (s *notificationStub) MarkAllRead(
	_ context.Context,
	_ auth.Principal,
) (int64, error) {
	s.markAll = true
	return 3, nil
}

func TestListNotificationsParsesFilters(t *testing.T) {
	service := &notificationStub{}
	handler := newNotificationTestHandler(service)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/me/notifications?page=2&pageSize=10&unreadOnly=true",
		nil,
	)
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	if service.listInput.Page != 2 || service.listInput.PageSize != 10 || !service.listInput.UnreadOnly {
		t.Fatalf("unexpected list input: %+v", service.listInput)
	}
	var result notifications.ListResult
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.UnreadCount != 2 {
		t.Fatalf("expected unread count 2, got %d", result.UnreadCount)
	}
}

func TestListNotificationsRejectsUnsupportedQuery(t *testing.T) {
	handler := newNotificationTestHandler(&notificationStub{})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/notifications?search=x", nil)
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", response.Code)
	}
}

func TestMarkNotificationsRead(t *testing.T) {
	service := &notificationStub{}
	handler := newNotificationTestHandler(service)
	id := "00000000-0000-4000-8000-000000000901"
	request := httptest.NewRequest(http.MethodPost, "/api/v1/me/notifications/"+id+"/read", nil)
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || service.markedID != id {
		t.Fatalf("mark read failed: status=%d id=%s", response.Code, service.markedID)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/me/notifications/read-all", nil)
	request.Header.Set("Authorization", "Bearer valid")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !service.markAll {
		t.Fatalf("mark all read failed: status=%d", response.Code)
	}
}

func newNotificationTestHandler(service notificationService) http.Handler {
	return New(Dependencies{
		AllowedOrigin: "http://localhost:4200",
		Logger:        slog.New(slog.NewTextHandler(discardWriter{}, nil)),
		Notifications: service,
		TokenVerifier: verifierStub{},
	})
}
