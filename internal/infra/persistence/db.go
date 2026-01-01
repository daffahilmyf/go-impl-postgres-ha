package persistence

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/daffahilmyf/go-impl-postgres-ha/internal/domain/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

type Config struct {
	WriteDSN          string
	ReadDSN           string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

type DB struct {
	Conn *gorm.DB
}

var _ repository.Store = (*DB)(nil)

type txKey struct{}

func New(ctx context.Context, cfg Config) (*DB, error) {
	if cfg.WriteDSN == "" {
		return nil, errors.New("db: WriteDSN is required")
	}

	writeDSN := normalizeDSN(cfg.WriteDSN)
	writeDialector := postgres.New(postgres.Config{
		DSN:                  writeDSN,
		PreferSimpleProtocol: true,
	})
	gdb, err := gorm.Open(writeDialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	readDSNs := splitDSNs(cfg.ReadDSN)
	for i := range readDSNs {
		readDSNs[i] = normalizeDSN(readDSNs[i])
	}
	if len(readDSNs) > 0 && !sameDSNs(readDSNs, writeDSN) {
		replicas := make([]gorm.Dialector, 0, len(readDSNs))
		for _, dsn := range readDSNs {
			replicas = append(replicas, postgres.New(postgres.Config{
				DSN:                  dsn,
				PreferSimpleProtocol: true,
			}))
		}

		resolverCfg := dbresolver.Config{
			Sources:  []gorm.Dialector{writeDialector},
			Replicas: replicas,
			Policy:   dbresolver.RandomPolicy{},
		}
		if err := gdb.Use(dbresolver.Register(resolverCfg).
			SetMaxOpenConns(int(cfg.MaxConns)).
			SetMaxIdleConns(int(cfg.MinConns)).
			SetConnMaxLifetime(cfg.MaxConnLifetime).
			SetConnMaxIdleTime(cfg.MaxConnIdleTime),
		); err != nil {
			return nil, err
		}
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	if cfg.MaxConns > 0 {
		sqlDB.SetMaxOpenConns(int(cfg.MaxConns))
	}
	if cfg.MinConns > 0 {
		sqlDB.SetMaxIdleConns(int(cfg.MinConns))
	}
	if cfg.MaxConnLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.MaxConnLifetime)
	}
	if cfg.MaxConnIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.MaxConnIdleTime)
	}

	return &DB{Conn: gdb}, nil
}

func (db *DB) Close() {
	if db == nil || db.Conn == nil {
		return
	}
	sqlDB, err := db.Conn.DB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
}

func (db *DB) Ping(ctx context.Context) error {
	if db == nil || db.Conn == nil {
		return errors.New("db: gorm connection is not initialized")
	}
	sqlDB, err := db.Conn.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (db *DB) Write(ctx context.Context) *gorm.DB {
	return db.getConn(ctx)
}

func (db *DB) Read(ctx context.Context) *gorm.DB {
	if db == nil || db.Conn == nil {
		return nil
	}
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return db.Conn.WithContext(ctx).Clauses(dbresolver.Read)
}

func (db *DB) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if db == nil || db.Conn == nil {
		return errors.New("db: gorm connection is not initialized")
	}
	return db.Conn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		return fn(txCtx)
	})
}

func (db *DB) getConn(ctx context.Context) *gorm.DB {
	if db == nil || db.Conn == nil {
		return nil
	}
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return db.Conn.WithContext(ctx)
}

func splitDSNs(input string) []string {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func sameDSNs(readDSNs []string, writeDSN string) bool {
	if len(readDSNs) == 0 {
		return true
	}
	for _, dsn := range readDSNs {
		if dsn != writeDSN {
			return false
		}
	}
	return true
}

func normalizeDSN(dsn string) string {
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Scheme == "" {
		return dsn
	}
	q := parsed.Query()
	if q.Get("statement_cache_capacity") == "" {
		q.Set("statement_cache_capacity", "0")
	}
	if q.Get("default_query_exec_mode") == "" {
		q.Set("default_query_exec_mode", "simple_protocol")
	}
	parsed.RawQuery = q.Encode()
	return parsed.String()
}
