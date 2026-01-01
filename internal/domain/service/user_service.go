package service

import (
	"context"

	"github.com/daffahilmyf/go-impl-postgres-ha/internal/domain/entity"
	"github.com/google/uuid"
)

type UserService interface {
	Create(ctx context.Context, name, email, idempotencyKey, requestHash string) (entity.User, bool, error)
	GetByID(ctx context.Context, id uuid.UUID) (entity.User, error)
	Update(ctx context.Context, id uuid.UUID, name, email string) (entity.User, error)
	DeleteByID(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit int, cursor string) ([]entity.User, string, error)
}
