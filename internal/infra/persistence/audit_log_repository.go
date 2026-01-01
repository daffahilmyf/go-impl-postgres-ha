package persistence

import (
	"context"
	"time"

	"github.com/daffahilmyf/go-impl-postgres-ha/internal/domain/entity"
	"gorm.io/datatypes"
)

type AuditLogRepository struct {
	db *DB
}

func NewAuditLogRepository(db *DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, eventType string, payload []byte) error {
	log := entity.AuditLog{
		EventType: eventType,
		Payload:   datatypes.JSON(payload),
		CreatedAt: time.Now().UTC(),
	}
	return r.db.Write(ctx).Create(&log).Error
}
