package config

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Asynq    AsynqConfig    `mapstructure:"asynq"`
	Python   PythonConfig   `mapstructure:"python"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Paths    PathsConfig    `mapstructure:"paths"`
}

type ServerConfig struct {
	Port       string `mapstructure:"port"`
	Mode       string `mapstructure:"mode"`
	WebDistDir string `mapstructure:"web_dist_dir"`
}

type DatabaseConfig struct {
	DSN                   string `mapstructure:"dsn"`
	URL                   string `mapstructure:"url"`
	MaxOpenConns          int    `mapstructure:"max_open_conns"`
	MaxIdleConns          int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetimeSecond int    `mapstructure:"conn_max_lifetime_sec"`
	ConnMaxIdleTimeSecond int    `mapstructure:"conn_max_idle_time_sec"`
	AutoMigrate           bool   `mapstructure:"auto_migrate"`
	Reset                 bool   `mapstructure:"reset"`
}

func (c DatabaseConfig) ConnectionString() string {
	if trimmed := strings.TrimSpace(c.DSN); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(c.URL)
}

func (c DatabaseConfig) ConnMaxLifetime() time.Duration {
	if c.ConnMaxLifetimeSecond <= 0 {
		return 0
	}
	return time.Duration(c.ConnMaxLifetimeSecond) * time.Second
}

func (c DatabaseConfig) ConnMaxIdleTime() time.Duration {
	if c.ConnMaxIdleTimeSecond <= 0 {
		return 0
	}
	return time.Duration(c.ConnMaxIdleTimeSecond) * time.Second
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type AsynqConfig struct {
	RedisAddr   string `mapstructure:"redis_addr"`
	Concurrency int    `mapstructure:"concurrency"`
}

func (c AsynqConfig) EffectiveRedisAddr(fallback string) string {
	if trimmed := strings.TrimSpace(c.RedisAddr); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(fallback)
}

type PythonConfig struct {
	Bin            string `mapstructure:"bin"`
	Script         string `mapstructure:"script"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
}

func (c PythonConfig) Timeout() time.Duration {
	if c.TimeoutSeconds <= 0 {
		return 60 * time.Second
	}
	return time.Duration(c.TimeoutSeconds) * time.Second
}

type LoggingConfig struct {
	File  string `mapstructure:"file"`
	Level string `mapstructure:"level"`
}

type PathsConfig struct {
	OctoModuleDir string `mapstructure:"octo_module_dir"`
}

func Load() (Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	v.SetDefault("server.port", "8080")
	v.SetDefault("server.mode", "release")
	v.SetDefault("server.web_dist_dir", "")

	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.db", 0)

	v.SetDefault("database.dsn", "")
	v.SetDefault("database.url", "")
	v.SetDefault("database.auto_migrate", true)
	v.SetDefault("database.reset", false)

	v.SetDefault("asynq.concurrency", 10)

	v.SetDefault("python.bin", "python")
	v.SetDefault("python.script", "../scripts/python/account_manager.py")
	v.SetDefault("python.timeout_seconds", 60)

	v.SetDefault("logging.file", "logs/octomanger.log")
	v.SetDefault("logging.level", "info")

	v.SetDefault("paths.octo_module_dir", "../scripts/python/modules")

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	_ = v.BindEnv("database.dsn", "DATABASE_DSN")
	_ = v.BindEnv("database.url", "DATABASE_URL")

	configFile := strings.TrimSpace(os.Getenv("CONFIG_FILE"))
	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		v.SetConfigName("config")
		v.AddConfigPath(".")
		v.AddConfigPath("configs")
		v.AddConfigPath("..")
		v.AddConfigPath("../configs")
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return Config{}, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func MustLoad() Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}
