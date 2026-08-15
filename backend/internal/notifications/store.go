package notifications

import (
	"context"
	"errors"
	"fmt"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/dx-os-lab/dx-os/backend/internal/platform/identity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	database *pgxpool.Pool
}

func NewStore(database *pgxpool.Pool) *Store {
	return &Store{database: database}
}

func Queue(ctx context.Context, tx pgx.Tx, input QueueInput) error {
	normalizeQueueInput(&input)
	if input.EventType == "" || input.ResourceType == "" || input.ResourceID == "" ||
		input.OrganizationID == "" || input.ActorID == "" || input.Title == "" || input.Body == "" ||
		(input.RecipientUserID == "" && input.RecipientRole == "") {
		return errors.New("notification outbox input is incomplete")
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO outbox_events (
			event_type, resource_type, resource_id, organization_id, department_id,
			recipient_user_id, recipient_role, actor_id, title, body
		)
		VALUES (
			$1, $2, $3, $4, NULLIF($5, '')::uuid,
			NULLIF($6, '')::uuid, NULLIF($7, ''), $8, $9, $10
		)
	`, input.EventType, input.ResourceType, input.ResourceID, input.OrganizationID,
		input.DepartmentID, input.RecipientUserID, input.RecipientRole, input.ActorID,
		input.Title, input.Body)
	if err != nil {
		return fmt.Errorf("queue notification outbox event: %w", err)
	}
	return nil
}

func (s *Store) List(
	ctx context.Context,
	principal auth.Principal,
	input ListInput,
) (ListResult, error) {
	ValidateListInput(&input)
	user, err := s.profile(ctx, principal)
	if err != nil {
		return ListResult{}, err
	}
	result := ListResult{
		Items: make([]Notification, 0), Page: input.Page, PageSize: input.PageSize,
	}
	offset := (input.Page - 1) * input.PageSize
	rows, err := s.database.Query(ctx, `
		SELECT
			n.id, n.event_type, n.resource_type, n.resource_id, n.title, n.body,
			n.created_at, nr.read_at,
			count(*) OVER(),
			count(*) FILTER (WHERE nr.read_at IS NULL) OVER()
		FROM user_notifications n
		LEFT JOIN notification_reads nr
			ON nr.notification_id = n.id AND nr.user_id = $1
		WHERE n.organization_id = $2
		  AND n.actor_id <> $1
		  AND (
			n.recipient_user_id = $1 OR (
				n.recipient_role = ANY($3::text[])
				AND (n.department_id IS NULL OR n.department_id = $4)
			)
		  )
		  AND (NOT $5::boolean OR nr.read_at IS NULL)
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT $6 OFFSET $7
	`, user.ID, user.OrganizationID, principal.Roles, user.DepartmentID,
		input.UnreadOnly, input.PageSize, offset)
	if err != nil {
		return ListResult{}, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item Notification
		if err = rows.Scan(
			&item.ID, &item.EventType, &item.ResourceType, &item.ResourceID,
			&item.Title, &item.Body, &item.CreatedAt, &item.ReadAt,
			&result.Total, &result.UnreadCount,
		); err != nil {
			return ListResult{}, fmt.Errorf("scan notification: %w", err)
		}
		result.Items = append(result.Items, item)
	}
	if err = rows.Err(); err != nil {
		return ListResult{}, fmt.Errorf("iterate notifications: %w", err)
	}
	if result.Total > 0 {
		result.Pages = int((result.Total + int64(input.PageSize) - 1) / int64(input.PageSize))
	}
	return result, nil
}

func (s *Store) MarkRead(
	ctx context.Context,
	principal auth.Principal,
	notificationID string,
) error {
	user, err := s.profile(ctx, principal)
	if err != nil {
		return err
	}
	command, err := s.database.Exec(ctx, `
		INSERT INTO notification_reads (notification_id, user_id)
		SELECT n.id, $1
		FROM user_notifications n
		WHERE n.id = $2
		  AND n.organization_id = $3
		  AND n.actor_id <> $1
		  AND (
			n.recipient_user_id = $1 OR (
				n.recipient_role = ANY($4::text[])
				AND (n.department_id IS NULL OR n.department_id = $5)
			)
		  )
		ON CONFLICT (notification_id, user_id) DO NOTHING
	`, user.ID, notificationID, user.OrganizationID, principal.Roles, user.DepartmentID)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	if command.RowsAffected() == 0 {
		var visible bool
		err = s.database.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM user_notifications n
				WHERE n.id = $1 AND n.organization_id = $2 AND n.actor_id <> $3
				  AND (n.recipient_user_id = $3 OR (
					n.recipient_role = ANY($4::text[])
					AND (n.department_id IS NULL OR n.department_id = $5)
				  ))
			)
		`, notificationID, user.OrganizationID, user.ID, principal.Roles, user.DepartmentID).Scan(&visible)
		if err != nil {
			return fmt.Errorf("verify notification scope: %w", err)
		}
		if !visible {
			return ErrNotFound
		}
	}
	return nil
}

func (s *Store) MarkAllRead(ctx context.Context, principal auth.Principal) (int64, error) {
	user, err := s.profile(ctx, principal)
	if err != nil {
		return 0, err
	}
	command, err := s.database.Exec(ctx, `
		INSERT INTO notification_reads (notification_id, user_id)
		SELECT n.id, $1
		FROM user_notifications n
		LEFT JOIN notification_reads nr
			ON nr.notification_id = n.id AND nr.user_id = $1
		WHERE n.organization_id = $2
		  AND n.actor_id <> $1
		  AND nr.notification_id IS NULL
		  AND (n.recipient_user_id = $1 OR (
			n.recipient_role = ANY($3::text[])
			AND (n.department_id IS NULL OR n.department_id = $4)
		  ))
		ON CONFLICT (notification_id, user_id) DO NOTHING
	`, user.ID, user.OrganizationID, principal.Roles, user.DepartmentID)
	if err != nil {
		return 0, fmt.Errorf("mark all notifications read: %w", err)
	}
	return command.RowsAffected(), nil
}

func (s *Store) profile(ctx context.Context, principal auth.Principal) (identity.Profile, error) {
	user, err := identity.Ensure(ctx, s.database, principal, "GENERAL")
	if errors.Is(err, identity.ErrInactive) {
		return identity.Profile{}, ErrForbidden
	}
	if err != nil {
		return identity.Profile{}, fmt.Errorf("load notification user profile: %w", err)
	}
	return user, nil
}
