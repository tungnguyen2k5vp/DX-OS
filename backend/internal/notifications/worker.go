package notifications

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Worker struct {
	database *pgxpool.Pool
}

func NewWorker(database *pgxpool.Pool) *Worker {
	return &Worker{database: database}
}

func (w *Worker) ProcessBatch(ctx context.Context, batchSize int) (int64, error) {
	if batchSize < 1 || batchSize > 200 {
		batchSize = 50
	}
	tx, err := w.database.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin outbox batch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	command, err := tx.Exec(ctx, `
		WITH claimed AS (
			SELECT *
			FROM outbox_events
			WHERE status = 'PENDING' AND next_attempt_at <= now()
			ORDER BY created_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		), materialized AS (
			INSERT INTO user_notifications (
				outbox_event_id, event_type, resource_type, resource_id,
				organization_id, department_id, recipient_user_id, recipient_role,
				actor_id, title, body, created_at
			)
			SELECT
				id, event_type, resource_type, resource_id,
				organization_id, department_id, recipient_user_id, recipient_role,
				actor_id, title, body, created_at
			FROM claimed
			ON CONFLICT (outbox_event_id) DO NOTHING
		)
		UPDATE outbox_events o
		SET status = 'PROCESSED', processed_at = now(), attempts = o.attempts + 1,
			last_error = NULL
		FROM claimed c
		WHERE o.id = c.id
	`, batchSize)
	if err != nil {
		return 0, fmt.Errorf("materialize outbox batch: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit outbox batch: %w", err)
	}
	return command.RowsAffected(), nil
}
