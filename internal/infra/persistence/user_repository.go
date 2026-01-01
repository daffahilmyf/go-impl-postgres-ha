package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/daffahilmyf/go-impl-postgres-ha/internal/domain/entity"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/domain/repository"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/infra/pagination"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *DB
}

func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, name, email string) (entity.User, error) {
	var user entity.User
	err := r.db.WithTx(ctx, func(txCtx context.Context) error {
		created, err := r.createUserWithOutbox(txCtx, name, email)
		if err != nil {
			return err
		}
		user = created
		return nil
	})
	if err != nil {
		return entity.User{}, err
	}
	return user, nil
}

func (r *UserRepository) CreateIdempotent(ctx context.Context, name, email, key, requestHash string) (entity.User, bool, error) {
	var (
		user         entity.User
		alreadyExist bool
	)
	err := r.db.WithTx(ctx, func(txCtx context.Context) error {
		var existing entity.IdempotencyKey
		if err := r.db.Write(txCtx).First(&existing, "key = ?", key).Error; err == nil {
			if existing.RequestHash != requestHash {
				return repository.ErrIdempotencyKeyConflict
			}
			fetched, err := r.GetByID(txCtx, existing.UserID)
			if err != nil {
				return err
			}
			user = fetched
			alreadyExist = true
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		created, err := r.createUserWithOutbox(txCtx, name, email)
		if err != nil {
			return err
		}
		user = created

		keyRow := entity.IdempotencyKey{
			Key:         key,
			RequestHash: requestHash,
			UserID:      user.ID,
		}
		if err := r.db.Write(txCtx).Create(&keyRow).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				var existing entity.IdempotencyKey
				if err := r.db.Write(txCtx).First(&existing, "key = ?", key).Error; err != nil {
					return err
				}
				if existing.RequestHash != requestHash {
					return repository.ErrIdempotencyKeyConflict
				}
				fetched, err := r.GetByID(txCtx, existing.UserID)
				if err != nil {
					return err
				}
				user = fetched
				alreadyExist = true
				return nil
			}
			return err
		}
		return nil
	})
	if err != nil {
		return entity.User{}, false, err
	}
	return user, alreadyExist, nil
}

func (r *UserRepository) createUserWithOutbox(ctx context.Context, name, email string) (entity.User, error) {
	user := entity.User{Name: name, Email: email}
	if err := r.db.Write(ctx).Create(&user).Error; err != nil {
		return entity.User{}, err
	}

	payload := struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"created_at"`
	}{
		ID:        user.ID.String(),
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return entity.User{}, err
	}

	outbox := entity.OutboxEvent{
		AggregateType: "user",
		AggregateID:   user.ID,
		EventType:     "user.created",
		Payload:       datatypes.JSON(data),
		CreatedAt:     time.Now().UTC(),
	}
	if err := r.db.Write(ctx).Create(&outbox).Error; err != nil {
		return entity.User{}, err
	}

	return user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (entity.User, error) {
	var user entity.User
	if err := r.db.Read(ctx).First(&user, "id = ?", id).Error; err != nil {
		return entity.User{}, err
	}
	return user, nil
}

func (r *UserRepository) Update(ctx context.Context, id uuid.UUID, name, email string) (entity.User, error) {
	if err := r.db.Write(ctx).
		Model(&entity.User{}).
		Where("id = ?", id).
		Updates(map[string]any{"name": name, "email": email}).Error; err != nil {
		return entity.User{}, err
	}
	return r.GetByID(ctx, id)
}

func (r *UserRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	return r.db.Write(ctx).Delete(&entity.User{}, "id = ?", id).Error
}

func (r *UserRepository) ListCursor(ctx context.Context, limit int, cursor string) ([]entity.User, error) {
	var users []entity.User
	if limit <= 0 {
		limit = 50
	}

	query := r.db.Read(ctx).
		Limit(limit).
		Order("created_at DESC").
		Order("id DESC")

	if cursor != "" {
		cursorTime, cursorID, err := pagination.Decode(cursor)
		if err != nil {
			if errors.Is(err, pagination.ErrInvalidCursor) {
				return nil, repository.ErrInvalidCursor
			}
			return nil, err
		}
		query = query.Where("(created_at < ?) OR (created_at = ? AND id < ?)", cursorTime, cursorTime, cursorID)
	}

	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
