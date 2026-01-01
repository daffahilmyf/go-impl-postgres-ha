package repository

import "context"

type Store interface {
	Ping(ctx context.Context) error
	Close()
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
