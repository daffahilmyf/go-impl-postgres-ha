package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Database struct {
	WriteDSN          string        `mapstructure:"write_dsn"`
	ReadDSN           string        `mapstructure:"read_dsn"`
	ConnectTimeout    time.Duration `mapstructure:"connect_timeout"`
	MaxConns          int32         `mapstructure:"max_conns"`
	MinConns          int32         `mapstructure:"min_conns"`
	MaxConnLifetime   time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `mapstructure:"max_conn_idle_time"`
	HealthCheckPeriod time.Duration `mapstructure:"health_check_period"`
}

type Config struct {
	Database Database `mapstructure:"database"`
	Server   Server   `mapstructure:"server"`
	Log      Log      `mapstructure:"log"`
	NATS     NATS     `mapstructure:"nats"`
	Outbox   Outbox   `mapstructure:"outbox"`
	Env      string   `mapstructure:"environment"`
}

type Server struct {
	Address      string        `mapstructure:"address"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

type Log struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type NATS struct {
	URL                string          `mapstructure:"url"`
	Stream             string          `mapstructure:"stream"`
	UserCreatedSubject string          `mapstructure:"user_created_subject"`
	DLQSubject         string          `mapstructure:"dlq_subject"`
	ConsumerDurable    string          `mapstructure:"consumer_durable"`
	AckWait            time.Duration   `mapstructure:"ack_wait"`
	MaxAckPending      int             `mapstructure:"max_ack_pending"`
	ConsumerMaxDeliver int             `mapstructure:"consumer_max_deliver"`
	ConsumerBackoff    []time.Duration `mapstructure:"consumer_backoff"`
}

type Outbox struct {
	BatchSize    int           `mapstructure:"batch_size"`
	PollInterval time.Duration `mapstructure:"poll_interval"`
	LockTimeout  time.Duration `mapstructure:"lock_timeout"`
	MaxAttempts  int           `mapstructure:"max_attempts"`
}

func Load(cfgFile string) (Config, error) {
	v := viper.New()

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName("config")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.go-impl-postgres-ha")
		v.AddConfigPath("/etc/go-impl-postgres-ha")
	}

	v.SetEnvPrefix("GO_IMPL_POSTGRES_HA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("database.max_conns", 20)
	v.SetDefault("database.min_conns", 0)
	v.SetDefault("database.connect_timeout", "5s")
	v.SetDefault("database.max_conn_lifetime", "30m")
	v.SetDefault("database.max_conn_idle_time", "5m")
	v.SetDefault("database.health_check_period", "1m")
	v.SetDefault("server.address", ":8080")
	v.SetDefault("server.read_timeout", "5s")
	v.SetDefault("server.write_timeout", "10s")
	v.SetDefault("server.idle_timeout", "60s")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")
	v.SetDefault("nats.stream", "events")
	v.SetDefault("nats.user_created_subject", "user.created")
	v.SetDefault("nats.dlq_subject", "user.created.dlq")
	v.SetDefault("nats.consumer_durable", "user-created-worker")
	v.SetDefault("nats.ack_wait", "30s")
	v.SetDefault("nats.max_ack_pending", 256)
	v.SetDefault("nats.consumer_max_deliver", 10)
	v.SetDefault("outbox.batch_size", 100)
	v.SetDefault("outbox.poll_interval", "2s")
	v.SetDefault("outbox.lock_timeout", "60s")
	v.SetDefault("outbox.max_attempts", 10)
	v.SetDefault("environment", "dev")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok || cfgFile != "" {
			return Config{}, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
