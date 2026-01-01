package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type OutboxEvent struct {
	ID            uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	AggregateType string         `gorm:"not null"`
	AggregateID   uuid.UUID      `gorm:"type:uuid;not null"`
	EventType     string         `gorm:"not null"`
	Payload       datatypes.JSON `gorm:"type:jsonb;not null"`
	CreatedAt     time.Time      `gorm:"not null"`
	LockedAt      *time.Time     `gorm:""`
	ProcessedAt   *time.Time     `gorm:""`
	Attempts      int            `gorm:"not null;default:0"`
	LastError     string         `gorm:""`
}

func (OutboxEvent) TableName() string {
	return "outbox_events"
}
