/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/daffahilmyf/go-impl-postgres-ha/internal/bootstrap"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/config"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/infra/messaging"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/infra/persistence"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var consumerCmd = &cobra.Command{
	Use:   "consumer",
	Short: "Run a JetStream consumer for user.created events",
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

		client, err := messaging.NewNATS(cmd.Context(), cfg.NATS)
		if err != nil {
			fmt.Fprintln(os.Stderr, "nats error:", err)
			os.Exit(1)
		}
		if client == nil {
			fmt.Fprintln(os.Stderr, "nats error: nats url is required")
			os.Exit(1)
		}
		defer client.Close()

		js := client.JetStream()
		if js == nil {
			fmt.Fprintln(os.Stderr, "nats error: jetstream not initialized")
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
		auditRepo := persistence.NewAuditLogRepository(db)

		if err := ensureConsumer(cmd.Context(), cfg, js); err != nil {
			fmt.Fprintln(os.Stderr, "consumer config error:", err)
			os.Exit(1)
		}

		log.Infof("consumer: listening on %s (durable=%s)", cfg.NATS.UserCreatedSubject, cfg.NATS.ConsumerDurable)
		sub, err := js.PullSubscribe(
			cfg.NATS.UserCreatedSubject,
			cfg.NATS.ConsumerDurable,
			nats.Bind(cfg.NATS.Stream, cfg.NATS.ConsumerDurable),
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, "subscribe error:", err)
			os.Exit(1)
		}

		for {
			select {
			case <-cmd.Context().Done():
				return
			default:
			}

			msgs, err := sub.Fetch(50, nats.MaxWait(2*time.Second))
			if err != nil {
				if errors.Is(err, nats.ErrTimeout) {
					continue
				}
				log.WithError(err).Warn("consumer: fetch failed")
				continue
			}
			for _, msg := range msgs {
				if err := auditRepo.Create(cmd.Context(), msg.Subject, msg.Data); err != nil {
					log.WithError(err).Warn("consumer: audit log insert failed")
					handleConsumerError(cmd.Context(), cfg, client, msg, log)
					continue
				}
				log.Infof("user.created: %s", string(msg.Data))
				_ = msg.Ack()
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(consumerCmd)
}

func ensureConsumer(ctx context.Context, cfg config.Config, js nats.JetStreamContext) error {
	if cfg.NATS.Stream == "" {
		return errors.New("nats stream is required")
	}
	if cfg.NATS.ConsumerDurable == "" {
		return errors.New("nats consumer durable is required")
	}
	if cfg.NATS.UserCreatedSubject == "" {
		return errors.New("nats user created subject is required")
	}

	info, err := js.ConsumerInfo(cfg.NATS.Stream, cfg.NATS.ConsumerDurable, nats.Context(ctx))
	if err != nil && !errors.Is(err, nats.ErrConsumerNotFound) {
		return err
	}

	backoff := cfg.NATS.ConsumerBackoff
	maxDeliver := cfg.NATS.ConsumerMaxDeliver
	if maxDeliver <= 0 {
		maxDeliver = -1
	}

	if info != nil {
		if info.Config.MaxDeliver != maxDeliver || !sameBackoff(info.Config.BackOff, backoff) {
			if err := js.DeleteConsumer(cfg.NATS.Stream, cfg.NATS.ConsumerDurable, nats.Context(ctx)); err != nil {
				return err
			}
			info = nil
		}
	}

	if info == nil {
		consumerCfg := &nats.ConsumerConfig{
			Durable:       cfg.NATS.ConsumerDurable,
			AckPolicy:     nats.AckExplicitPolicy,
			AckWait:       cfg.NATS.AckWait,
			MaxAckPending: cfg.NATS.MaxAckPending,
			MaxDeliver:    maxDeliver,
			FilterSubject: cfg.NATS.UserCreatedSubject,
		}
		if len(backoff) > 0 {
			consumerCfg.BackOff = backoff
		}
		if _, err := js.AddConsumer(cfg.NATS.Stream, consumerCfg, nats.Context(ctx)); err != nil {
			return err
		}
	}
	return nil
}

func sameBackoff(a, b []time.Duration) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func handleConsumerError(ctx context.Context, cfg config.Config, client *messaging.NATSClient, msg *nats.Msg, log *logrus.Logger) {
	md, err := msg.Metadata()
	if err != nil {
		log.WithError(err).Warn("consumer: metadata missing")
		_ = msg.Nak()
		return
	}
	maxDeliver := cfg.NATS.ConsumerMaxDeliver
	if maxDeliver <= 0 {
		maxDeliver = 10
	}
	if int(md.NumDelivered) >= maxDeliver {
		if cfg.NATS.DLQSubject != "" {
			if err := client.Publish(ctx, cfg.NATS.DLQSubject, msg.Data, fmt.Sprintf("dlq-%d", md.Sequence.Stream)); err != nil {
				log.WithError(err).Warn("consumer: dlq publish failed")
				_ = msg.Nak()
				return
			}
		} else {
			log.Warn("consumer: dlq subject not configured")
		}
		_ = msg.Ack()
		return
	}
	delay := backoffForAttempt(cfg.NATS.ConsumerBackoff, md.NumDelivered)
	if delay > 0 {
		_ = msg.NakWithDelay(delay)
		return
	}
	_ = msg.Nak()
}

func backoffForAttempt(backoff []time.Duration, delivered uint64) time.Duration {
	if len(backoff) == 0 {
		return 0
	}
	idx := int(delivered) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(backoff) {
		idx = len(backoff) - 1
	}
	return backoff[idx]
}
