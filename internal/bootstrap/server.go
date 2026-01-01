package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/daffahilmyf/go-impl-postgres-ha/internal/config"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/infra/persistence"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/transport/http/handlers"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/transport/http/middleware"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func Run(ctx context.Context, cfg config.Config) error {
	start := time.Now()
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
	log.Infof("bootstrap: db init in %s", time.Since(start))
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
	log.Infof("bootstrap: db ping in %s", time.Since(start))

	userRepo := persistence.NewUserRepository(conn)
	userUC := usecase.NewUser(userRepo, log)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.Logger(log), gin.Recovery())
	allowBypassIdemKey := cfg.Env != "prod"
	handler := handlers.NewHandler(userUC, conn)
	routerBuilder := handlers.NewRouter(handler)
	routerBuilder.RegisterRoutes(router, middleware.IdempotencyRequired(allowBypassIdemKey))

	srv := &http.Server{
		Addr:         cfg.Server.Address,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Infof("bootstrap: server listening on %s", cfg.Server.Address)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		log.WithError(err).Error("server error")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Error("server shutdown error")
	}

	return nil
}

func buildLogger(cfg config.Config) (*logrus.Logger, error) {
	log := logrus.New()
	level, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		return nil, err
	}
	log.SetLevel(level)
	switch cfg.Log.Format {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	case "console", "":
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		})
	default:
		return nil, errors.New("log format error: supported values are console or json")
	}
	return log, nil
}
