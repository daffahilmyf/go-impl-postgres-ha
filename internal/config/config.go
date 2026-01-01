package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Database struct {
	WriteDSN          string        `mapstructure:"write_dsn"`
	ReadDSN           string        `mapstructure:"read_dsn"`
	Host              string        `mapstructure:"host"`
	ReadHost          string        `mapstructure:"read_host"`
	Port              int           `mapstructure:"port"`
	Name              string        `mapstructure:"name"`
	User              string        `mapstructure:"user"`
	Password          string        `mapstructure:"password"`
	SSLMode           string        `mapstructure:"sslmode"`
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
	_ = v.BindEnv("database.user", "DB_USER")
	_ = v.BindEnv("database.password", "DB_PASS")

	v.SetDefault("database.max_conns", 20)
	v.SetDefault("database.min_conns", 0)
	v.SetDefault("database.connect_timeout", "5s")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.sslmode", "disable")
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

	cfg = applyDSNDefaults(cfg)
	return cfg, nil
}

func applyDSNDefaults(cfg Config) Config {
	if cfg.Database.WriteDSN == "" && cfg.Database.Host != "" && cfg.Database.Name != "" {
		cfg.Database.WriteDSN = buildDSN(cfg.Database.Host, cfg.Database.Port, cfg.Database.Name, cfg.Database.User, cfg.Database.Password, cfg.Database.SSLMode)
	}
	if cfg.Database.ReadDSN == "" {
		readHost := cfg.Database.ReadHost
		if readHost == "" {
			readHost = cfg.Database.Host
		}
		if readHost != "" && cfg.Database.Name != "" {
			cfg.Database.ReadDSN = buildDSN(readHost, cfg.Database.Port, cfg.Database.Name, cfg.Database.User, cfg.Database.Password, cfg.Database.SSLMode)
		}
	}
	return cfg
}

func buildDSN(host string, port int, name, user, password, sslmode string) string {
	if sslmode == "" {
		sslmode = "disable"
	}
	creds := ""
	if user != "" {
		creds = user
		if password != "" {
			creds += ":" + password
		}
		creds += "@"
	}
	return "postgres://" + creds + host + ":" + fmt.Sprintf("%d", port) + "/" + name + "?sslmode=" + sslmode
}
