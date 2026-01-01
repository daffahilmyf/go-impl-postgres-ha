package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/daffahilmyf/go-impl-postgres-ha/internal/config"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/domain/entity"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/infra/persistence"
	"github.com/go-faker/faker/v4"
	"github.com/google/uuid"
)

func Seed(ctx context.Context, cfg config.Config, count, batchSize int) error {
	if count <= 0 {
		count = 10
	}
	if batchSize <= 0 {
		batchSize = 100
	}

	log, err := buildLogger(cfg)
	if err != nil {
		return err
	}

	conn, err := persistence.New(ctx, persistence.Config{
		WriteDSN:          cfg.Database.WriteDSN,
		ReadDSN:           cfg.Database.ReadDSN,
		MaxConns:          cfg.Database.MaxConns,
		MinConns:          cfg.Database.MinConns,
		MaxConnLifetime:   cfg.Database.MaxConnLifetime,
		MaxConnIdleTime:   cfg.Database.MaxConnIdleTime,
		HealthCheckPeriod: cfg.Database.HealthCheckPeriod,
	})
	if err != nil {
		return err
	}
	defer conn.Close()

	pingCtx := ctx
	if cfg.Database.ConnectTimeout > 0 {
		var cancel context.CancelFunc
		pingCtx, cancel = context.WithTimeout(ctx, cfg.Database.ConnectTimeout)
		defer cancel()
	}
	if err := conn.Ping(pingCtx); err != nil {
		return err
	}

	baseTime := time.Now().UTC()
	users := make([]entity.User, 0, batchSize)
	for i := 0; i < count; i++ {
		first := faker.FirstName()
		last := faker.LastName()
		seedTime := baseTime.Add(time.Duration(i) * time.Microsecond)
		user := entity.User{
			Name:      fmt.Sprintf("%s %s", first, last),
			Email:     fmt.Sprintf("seed-%s@example.com", uuid.NewString()),
			CreatedAt: seedTime,
			UpdatedAt: seedTime,
		}
		users = append(users, user)
		if len(users) == batchSize {
			if err := conn.Write(ctx).CreateInBatches(&users, batchSize).Error; err != nil {
				return err
			}
			users = users[:0]
		}
	}
	if len(users) > 0 {
		if err := conn.Write(ctx).CreateInBatches(&users, batchSize).Error; err != nil {
			return err
		}
	}

	log.Infof("bootstrap: seeded %d users", count)
	return nil
}
