/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/daffahilmyf/go-impl-postgres-ha/internal/bootstrap"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/config"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/infra/messaging"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/infra/persistence"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var outboxCmd = &cobra.Command{
	Use:   "outbox-worker",
	Short: "Publish outbox events to NATS JetStream",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "config error:", err)
			os.Exit(1)
		}
		log, err := bootstrap.BuildLogger(cfg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "log error:", err)
			os.Exit(1)
		}

		db, err := persistence.New(cmd.Context(), persistence.Config{
			WriteDSN:          cfg.Database.WriteDSN,
			ReadDSN:           cfg.Database.ReadDSN,
			MaxConns:          cfg.Database.MaxConns,
			MinConns:          cfg.Database.MinConns,
			MaxConnLifetime:   cfg.Database.MaxConnLifetime,
			MaxConnIdleTime:   cfg.Database.MaxConnIdleTime,
			HealthCheckPeriod: cfg.Database.HealthCheckPeriod,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "db error:", err)
			os.Exit(1)
		}
		defer db.Close()

		natsClient, err := messaging.NewNATS(cmd.Context(), cfg.NATS)
		if err != nil {
			fmt.Fprintln(os.Stderr, "nats error:", err)
			os.Exit(1)
		}
		if natsClient == nil {
			fmt.Fprintln(os.Stderr, "nats error: nats url is required")
			os.Exit(1)
		}
		defer natsClient.Close()

		repo := persistence.NewOutboxRepository(db)
		log.Infof("outbox-worker: started (batch=%d, interval=%s)", cfg.Outbox.BatchSize, cfg.Outbox.PollInterval)

		ticker := time.NewTicker(cfg.Outbox.PollInterval)
		defer ticker.Stop()

		for {
			if err := processOutbox(cmd.Context(), cfg, repo, natsClient, log); err != nil {
				log.WithError(err).Warn("outbox-worker: process failed")
			}
			select {
			case <-cmd.Context().Done():
				return
			case <-ticker.C:
			}
		}
	},
}

func processOutbox(ctx context.Context, cfg config.Config, repo *persistence.OutboxRepository, natsClient *messaging.NATSClient, log *logrus.Logger) error {
	events, err := repo.Claim(ctx, cfg.Outbox.BatchSize, cfg.Outbox.LockTimeout, cfg.Outbox.MaxAttempts)
	if err != nil {
		return err
	}
	for _, event := range events {
		subject := event.EventType
		if event.EventType == "user.created" {
			subject = cfg.NATS.UserCreatedSubject
		}
		if err := natsClient.Publish(ctx, subject, event.Payload, event.AggregateID.String()); err != nil {
			if err := repo.MarkFailed(ctx, event.ID, err.Error()); err != nil {
				log.WithError(err).Warn("outbox-worker: mark failed")
			}
			continue
		}
		if err := repo.MarkProcessed(ctx, event.ID); err != nil {
			log.WithError(err).Warn("outbox-worker: mark processed")
		}
	}
	return nil
}

func init() {
	rootCmd.AddCommand(outboxCmd)
}
