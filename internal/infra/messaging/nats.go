package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/daffahilmyf/go-impl-postgres-ha/internal/config"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/domain/entity"
	"github.com/nats-io/nats.go"
)

type NATSClient struct {
	conn *nats.Conn
	js   nats.JetStreamContext
	cfg  config.NATS
}

func NewNATS(ctx context.Context, cfg config.NATS) (*NATSClient, error) {
	if cfg.URL == "" {
		return nil, nil
	}
	if cfg.Stream == "" || cfg.UserCreatedSubject == "" {
		return nil, errors.New("nats: stream and user_created_subject are required")
	}

	conn, err := nats.Connect(cfg.URL, nats.Name("simple-backend"))
	if err != nil {
		return nil, err
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, err
	}

	if err := ensureStream(ctx, js, cfg); err != nil {
		conn.Close()
		return nil, err
	}

	return &NATSClient{conn: conn, js: js, cfg: cfg}, nil
}

func (c *NATSClient) Close() {
	if c == nil || c.conn == nil {
		return
	}
	c.conn.Close()
}

func (c *NATSClient) JetStream() nats.JetStreamContext {
	if c == nil {
		return nil
	}
	return c.js
}

func (c *NATSClient) PublishUserCreated(ctx context.Context, user entity.User) error {
	if c == nil {
		return nil
	}
	if c.js == nil {
		return errors.New("nats: jetstream not initialized")
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
		return err
	}

	return c.Publish(ctx, c.cfg.UserCreatedSubject, data, user.ID.String())
}

func (c *NATSClient) Publish(ctx context.Context, subject string, payload []byte, msgID string) error {
	if c == nil {
		return nil
	}
	if c.js == nil {
		return errors.New("nats: jetstream not initialized")
	}
	msg := nats.NewMsg(subject)
	msg.Data = payload
	if msgID != "" {
		msg.Header.Set(nats.MsgIdHdr, msgID)
	}
	_, err := c.js.PublishMsg(msg, nats.Context(ctx))
	return err
}

func ensureStream(ctx context.Context, js nats.JetStreamContext, cfg config.NATS) error {
	info, err := js.StreamInfo(cfg.Stream, nats.Context(ctx))
	if err == nil {
		subjects := []string{cfg.UserCreatedSubject}
		if cfg.DLQSubject != "" {
			subjects = append(subjects, cfg.DLQSubject)
		}
		if !sameSubjects(info.Config.Subjects, subjects) {
			info.Config.Subjects = subjects
			_, err = js.UpdateStream(&info.Config, nats.Context(ctx))
		}
		return err
	}

	if errors.Is(err, nats.ErrStreamNotFound) {
		subjects := []string{cfg.UserCreatedSubject}
		if cfg.DLQSubject != "" {
			subjects = append(subjects, cfg.DLQSubject)
		}
		_, err = js.AddStream(&nats.StreamConfig{
			Name:      cfg.Stream,
			Subjects:  subjects,
			Storage:   nats.FileStorage,
			Retention: nats.LimitsPolicy,
		}, nats.Context(ctx))
		return err
	}
	return err
}

func sameSubjects(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int, len(a))
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		if seen[s] == 0 {
			return false
		}
		seen[s]--
	}
	for _, v := range seen {
		if v != 0 {
			return false
		}
	}
	return true
}
