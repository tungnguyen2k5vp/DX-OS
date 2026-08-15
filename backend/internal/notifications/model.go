package notifications

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrForbidden = errors.New("notification access is forbidden")
	ErrNotFound  = errors.New("notification not found")
)

type QueueInput struct {
	EventType       string
	ResourceType    string
	ResourceID      string
	OrganizationID  string
	DepartmentID    string
	RecipientUserID string
	RecipientRole   string
	ActorID         string
	Title           string
	Body            string
}

type ListInput struct {
	Page       int
	PageSize   int
	UnreadOnly bool
}

type Notification struct {
	ID           string     `json:"id"`
	EventType    string     `json:"eventType"`
	ResourceType string     `json:"resourceType"`
	ResourceID   string     `json:"resourceId"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	CreatedAt    time.Time  `json:"createdAt"`
	ReadAt       *time.Time `json:"readAt"`
}

type ListResult struct {
	Items       []Notification `json:"items"`
	Page        int            `json:"page"`
	PageSize    int            `json:"pageSize"`
	Total       int64          `json:"total"`
	Pages       int            `json:"pages"`
	UnreadCount int64          `json:"unreadCount"`
}

func ValidateListInput(input *ListInput) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 {
		input.PageSize = 20
	}
	if input.PageSize > 50 {
		input.PageSize = 50
	}
}

func normalizeQueueInput(input *QueueInput) {
	input.EventType = strings.TrimSpace(input.EventType)
	input.ResourceType = strings.TrimSpace(input.ResourceType)
	input.ResourceID = strings.TrimSpace(input.ResourceID)
	input.OrganizationID = strings.TrimSpace(input.OrganizationID)
	input.DepartmentID = strings.TrimSpace(input.DepartmentID)
	input.RecipientUserID = strings.TrimSpace(input.RecipientUserID)
	input.RecipientRole = strings.TrimSpace(input.RecipientRole)
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.Title = strings.TrimSpace(input.Title)
	input.Body = strings.TrimSpace(input.Body)
}
