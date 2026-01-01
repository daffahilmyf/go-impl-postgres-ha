package persistence

import (
	"context"
	"time"

	"github.com/daffahilmyf/go-impl-postgres-ha/internal/domain/entity"
	"github.com/google/uuid"
)

type OutboxRepository struct {
	db *DB
}

func NewOutboxRepository(db *DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) Claim(ctx context.Context, limit int, lockTimeout time.Duration, maxAttempts int) ([]entity.OutboxEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	if lockTimeout <= 0 {
		lockTimeout = time.Minute
	}
	if maxAttempts <= 0 {
		maxAttempts = 10
	}
	lockSeconds := int(lockTimeout.Seconds())

	query := `
WITH cte AS (
    SELECT id
    FROM outbox_events
    WHERE processed_at IS NULL
      AND attempts < ?
      AND (locked_at IS NULL OR locked_at < NOW() - (? * INTERVAL '1 second'))
    ORDER BY created_at
    LIMIT ?
    FOR UPDATE SKIP LOCKED
)
UPDATE outbox_events
SET locked_at = NOW(), attempts = attempts + 1
WHERE id IN (SELECT id FROM cte)
RETURNING id, aggregate_type, aggregate_id, event_type, payload, created_at, locked_at, processed_at, attempts, last_error;
`

	var events []entity.OutboxEvent
	if err := r.db.Write(ctx).Raw(query, maxAttempts, lockSeconds, limit).Scan(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

func (r *OutboxRepository) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	return r.db.Write(ctx).
		Exec(`UPDATE outbox_events SET processed_at = NOW(), locked_at = NULL WHERE id = ?`, id).
		Error
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	return r.db.Write(ctx).
		Exec(`UPDATE outbox_events SET last_error = ?, locked_at = NULL WHERE id = ?`, errMsg, id).
		Error
}
