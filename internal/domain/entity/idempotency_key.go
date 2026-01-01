package entity

import (
	"time"

	"github.com/google/uuid"
)

type IdempotencyKey struct {
	Key         string    `gorm:"primaryKey"`
	RequestHash string    `gorm:"not null"`
	UserID      uuid.UUID `gorm:"type:uuid;not null"`
	CreatedAt   time.Time `gorm:"not null"`
}

func (IdempotencyKey) TableName() string {
	return "idempotency_keys"
}
