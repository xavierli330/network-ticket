package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config holds all application configuration sections.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Log      LogConfig      `mapstructure:"log"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Security SecurityConfig `mapstructure:"security"`
	Worker   WorkerConfig   `mapstructure:"worker"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// DatabaseConfig holds MySQL connection settings.
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

// DSN returns a MySQL data source name string.
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		d.User, d.Password, d.Host, d.Port, d.DBName,
	)
}

// LogConfig holds logging settings including file rotation.
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	FilePath   string `mapstructure:"file_path"`
	MaxSizeMB  int    `mapstructure:"max_size_mb"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAgeDays int    `mapstructure:"max_age_days"`
}

// JWTConfig holds JWT authentication settings.
type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

// SecurityConfig holds security-related settings.
type SecurityConfig struct {
	Nonce NonceConfig `mapstructure:"nonce"`
}

// NonceConfig holds nonce storage settings.
type NonceConfig struct {
	Backend string    `mapstructure:"backend"`
	TTL     string    `mapstructure:"ttl"`
	File    FileStore `mapstructure:"file"`
}

// FileStore holds file-based nonce storage settings.
type FileStore struct {
	Path string `mapstructure:"path"`
}

// WorkerConfig holds background worker pool settings.
type WorkerConfig struct {
	PoolSize          int           `mapstructure:"pool_size"`
	RetryMax          int           `mapstructure:"retry_max"`
	RetryBaseInterval time.Duration `mapstructure:"retry_base_interval"`
	RetryMaxInterval  time.Duration `mapstructure:"retry_max_interval"`
}

// Load reads configuration from the given YAML file and merges environment
// variables (prefixed with NT_, e.g. NT_SERVER_PORT overrides server.port).
func Load(path string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// Allow environment variable overrides: NT_SERVER_PORT => server.port
	v.SetEnvPrefix("NT")
	v.AutomaticEnv()
	v.SetTypeByDefaultValue(true)

	// Explicitly bind nested config keys to env vars so Unmarshal picks them up.
	_ = v.BindEnv("server.port", "NT_SERVER_PORT")
	_ = v.BindEnv("server.mode", "NT_SERVER_MODE")
	_ = v.BindEnv("database.host", "NT_DATABASE_HOST")
	_ = v.BindEnv("database.port", "NT_DATABASE_PORT")
	_ = v.BindEnv("database.user", "NT_DATABASE_USER")
	_ = v.BindEnv("database.password", "NT_DATABASE_PASSWORD")
	_ = v.BindEnv("database.dbname", "NT_DATABASE_DBNAME")
	_ = v.BindEnv("database.max_open_conns", "NT_DATABASE_MAX_OPEN_CONNS")
	_ = v.BindEnv("database.max_idle_conns", "NT_DATABASE_MAX_IDLE_CONNS")
	_ = v.BindEnv("log.level", "NT_LOG_LEVEL")
	_ = v.BindEnv("log.format", "NT_LOG_FORMAT")
	_ = v.BindEnv("log.file_path", "NT_LOG_FILE_PATH")
	_ = v.BindEnv("log.max_size_mb", "NT_LOG_MAX_SIZE_MB")
	_ = v.BindEnv("log.max_backups", "NT_LOG_MAX_BACKUPS")
	_ = v.BindEnv("log.max_age_days", "NT_LOG_MAX_AGE_DAYS")
	_ = v.BindEnv("jwt.secret", "NT_JWT_SECRET")
	_ = v.BindEnv("jwt.expire_hours", "NT_JWT_EXPIRE_HOURS")
	_ = v.BindEnv("security.nonce.backend", "NT_SECURITY_NONCE_BACKEND")
	_ = v.BindEnv("security.nonce.ttl", "NT_SECURITY_NONCE_TTL")
	_ = v.BindEnv("security.nonce.file.path", "NT_SECURITY_NONCE_FILE_PATH")
	_ = v.BindEnv("worker.pool_size", "NT_WORKER_POOL_SIZE")
	_ = v.BindEnv("worker.retry_max", "NT_WORKER_RETRY_MAX")
	_ = v.BindEnv("worker.retry_base_interval", "NT_WORKER_RETRY_BASE_INTERVAL")
	_ = v.BindEnv("worker.retry_max_interval", "NT_WORKER_RETRY_MAX_INTERVAL")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}

// InitLogger creates a zap.Logger with dual output (stdout + rotating file).
// Both writers use the same encoder determined by cfg.Format ("json" or "text"/"console").
func InitLogger(cfg *LogConfig) (*zap.Logger, error) {
	// Determine log level.
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	// Build encoder config.
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	// Choose encoder based on format.
	var encoder zapcore.Encoder
	switch cfg.Format {
	case "json":
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	default:
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	}

	// Stdout core.
	stdoutCore := zapcore.NewCore(
		encoder,
		zapcore.AddSync(zapcore.Lock(os.Stdout)),
		level,
	)

	cores := []zapcore.Core{stdoutCore}

	// File core with lumberjack rotation (only if file_path is set).
	if cfg.FilePath != "" {
		lj := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAgeDays,
		}
		fileCore := zapcore.NewCore(
			encoder,
			zapcore.AddSync(lj),
			level,
		)
		cores = append(cores, fileCore)
	}

	// Combine cores.
	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger, nil
}
