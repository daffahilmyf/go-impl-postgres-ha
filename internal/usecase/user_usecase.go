package usecase

import (
	"context"

	"github.com/daffahilmyf/go-impl-postgres-ha/internal/domain/entity"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/domain/repository"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/domain/service"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/infra/pagination"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type User struct {
	repo repository.UserRepository
	log  *logrus.Logger
}

var _ service.UserService = (*User)(nil)

func NewUser(repo repository.UserRepository, log *logrus.Logger) *User {
	return &User{repo: repo, log: log}
}

func (u *User) Create(ctx context.Context, name, email, idempotencyKey, requestHash string) (entity.User, bool, error) {
	if idempotencyKey == "" {
		user, err := u.repo.Create(ctx, name, email)
		if err != nil {
			u.log.WithError(err).Error("create user failed")
			return entity.User{}, false, err
		}
		return user, false, nil
	}

	user, alreadyExist, err := u.repo.CreateIdempotent(ctx, name, email, idempotencyKey, requestHash)
	if err != nil {
		u.log.WithError(err).Error("create user failed")
		return entity.User{}, false, err
	}
	return user, alreadyExist, nil
}

func (u *User) GetByID(ctx context.Context, id uuid.UUID) (entity.User, error) {
	user, err := u.repo.GetByID(ctx, id)
	if err != nil {
		u.log.WithError(err).Error("get user failed")
		return entity.User{}, err
	}
	return user, nil
}

func (u *User) Update(ctx context.Context, id uuid.UUID, name, email string) (entity.User, error) {
	user, err := u.repo.Update(ctx, id, name, email)
	if err != nil {
		u.log.WithError(err).Error("update user failed")
		return entity.User{}, err
	}
	return user, nil
}

func (u *User) DeleteByID(ctx context.Context, id uuid.UUID) error {
	if err := u.repo.DeleteByID(ctx, id); err != nil {
		u.log.WithError(err).Error("delete user failed")
		return err
	}
	return nil
}

func (u *User) List(ctx context.Context, limit int, cursor string) ([]entity.User, string, error) {
	users, err := u.repo.ListCursor(ctx, limit, cursor)
	if err != nil {
		u.log.WithError(err).Error("list users failed")
		return nil, "", err
	}
	nextCursor := ""
	if len(users) > 0 && (limit <= 0 || len(users) == limit) {
		last := users[len(users)-1]
		nextCursor = pagination.Encode(last.CreatedAt, last.ID)
	}
	return users, nextCursor, nil
}
