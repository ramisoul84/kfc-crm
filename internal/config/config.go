package config

import (
	"fmt"
	"strings"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	App    AppConfig
	HTTP   HTTPConfig
	Logger LoggerConfig
	DB     DatabaseConfig
	Redis  RedisConfig
	JWT    JWTConfig
}

// AppConfig holds application-level configuration
type AppConfig struct {
	Name        string
	Version     string
	Environment string
	Debug       bool
}

// HTTPConfig holds HTTP server configuration
type HTTPConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// LoggerConfig holds logger settings.
type LoggerConfig struct {
	Level    string
	Format   string
	Output   string
	FilePath string
	Service  string
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxConns        int
	MinConns        int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	ConnectTimeout  time.Duration
	Timezone        string
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Host         string
	Port         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret          string
	AccessDuration  time.Duration
	RefreshDuration time.Duration
}

// Load reads configuration from environment variables and validates it.
func Load() (*Config, error) {
	env := getEnv("APP_ENV", "development")
	loadDotenv(env)

	cfg := &Config{
		App: AppConfig{
			Name:        getEnv("APP_NAME", "kfc-crm"),
			Version:     getEnv("APP_VERSION", "1.0.0"),
			Environment: env,
			Debug:       getEnvBool("APP_DEBUG", env != "production"),
		},

		HTTP: HTTPConfig{
			Port:            getEnv("HTTP_PORT", "8000"),
			ReadTimeout:     getEnvDuration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getEnvDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     getEnvDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getEnvDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},

		Logger: LoggerConfig{
			Level:    getEnv("LOG_LEVEL", defaultLogLevel(env)),
			Format:   getEnv("LOG_FORMAT", defaultLogFormat(env)),
			Output:   getEnv("LOG_OUTPUT", defaultLogOutput(env)),
			FilePath: getEnv("LOG_FILE_PATH", "logs/app.log"),
			Service:  getEnv("LOG_SERVICE", "kfc-crm"),
		},

		DB: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			Name:            getEnv("DB_NAME", "kfc_crm"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxConns:        getEnvInt("DB_MAX_CONNS", 20),
			MinConns:        getEnvInt("DB_MIN_CONNS", 5),
			MaxConnLifetime: getEnvDuration("DB_MAX_CONN_LIFETIME", time.Hour),
			MaxConnIdleTime: getEnvDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
			ConnectTimeout:  getEnvDuration("DB_CONNECT_TIMEOUT", 10*time.Second),
			Timezone:        getEnv("DB_TIMEZONE", "Europe/Moscow"),
		},

		Redis: RedisConfig{
			Host:         getEnv("REDIS_HOST", "localhost"),
			Port:         getEnv("REDIS_PORT", "6379"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvInt("REDIS_DB", 0),
			PoolSize:     getEnvInt("REDIS_POOL_SIZE", 10),
			MinIdleConns: getEnvInt("REDIS_MIN_IDLE_CONNS", 5),
			DialTimeout:  getEnvDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:  getEnvDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout: getEnvDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
		},

		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "dev-secret-change-in-production"),
			AccessDuration:  getEnvDuration("JWT_ACCESS_DURATION", 15*time.Minute),
			RefreshDuration: getEnvDuration("JWT_REFRESH_DURATION", 7*24*time.Hour),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	var errors []string

	// Validate environment
	if !isValidEnvironment(c.App.Environment) {
		errors = append(errors, fmt.Sprintf("invalid APP_ENV: %q", c.App.Environment))
	}

	// Validate logger
	if !isValidLogLevel(c.Logger.Level) {
		errors = append(errors, fmt.Sprintf("invalid LOG_LEVEL: %q", c.Logger.Level))
	}
	if !isValidLogFormat(c.Logger.Format) {
		errors = append(errors, fmt.Sprintf("invalid LOG_FORMAT: %q", c.Logger.Format))
	}
	if !isValidLogOutput(c.Logger.Output) {
		errors = append(errors, fmt.Sprintf("invalid LOG_OUTPUT: %q", c.Logger.Output))
	}

	// Validate ports
	ports := map[string]string{
		"HTTP_PORT":  c.HTTP.Port,
		"DB_PORT":    c.DB.Port,
		"REDIS_PORT": c.Redis.Port,
	}

	for name, port := range ports {
		if !isValidPort(port) {
			errors = append(errors, fmt.Sprintf("invalid %s: %q", name, port))
		}
	}

	// Validate JWT
	if c.JWT.AccessDuration <= 0 {
		errors = append(errors, "JWT_ACCESS_DURATION must be positive")
	}
	if c.JWT.RefreshDuration <= 0 {
		errors = append(errors, "JWT_REFRESH_DURATION must be positive")
	}
	if c.JWT.AccessDuration >= c.JWT.RefreshDuration {
		errors = append(errors, "JWT_ACCESS_DURATION must be less than JWT_REFRESH_DURATION")
	}

	// Validate database pool
	if c.DB.MaxConns < c.DB.MinConns {
		errors = append(errors, "DB_MAX_CONNS must be greater than or equal to DB_MIN_CONNS")
	}

	// Validate Redis pool
	if c.Redis.PoolSize < c.Redis.MinIdleConns {
		errors = append(errors, "REDIS_POOL_SIZE must be greater than or equal to REDIS_MIN_IDLE_CONNS")
	}

	// Production security checks
	if c.App.Environment == "production" {
		if len(c.JWT.Secret) < 32 {
			errors = append(errors, "JWT_SECRET must be at least 32 characters in production")
		}
		if len(c.DB.Password) < 32 {
			errors = append(errors, "DB_PASSWORD must be secure in production")
		}
		if len(c.Redis.Password) < 32 {
			errors = append(errors, "REDIS_PASSWORD must be secure in production")
		}
		if c.DB.SSLMode != "require" && c.DB.SSLMode != "verify-full" {
			errors = append(errors, "DB_SSL_MODE should be 'require' or 'verify-full' in production")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return nil
}

// IsProduction returns true if environment is production
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}

// IsDevelopment returns true if environment is development
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development"
}

// Helper validation functions
func isValidEnvironment(env string) bool {
	switch env {
	case "development", "staging", "production":
		return true
	}
	return false
}

func isValidLogLevel(level string) bool {
	switch level {
	case "debug", "info", "warn", "error":
		return true
	}
	return false
}

func isValidLogFormat(format string) bool {
	return format == "json" || format == "text"
}

func isValidLogOutput(output string) bool {
	switch output {
	case "stdout", "file", "both":
		return true
	}
	return false
}

func defaultLogLevel(env string) string {
	if env == "production" {
		return "info"
	}
	return "debug"
}

func defaultLogFormat(env string) string {
	if env == "production" {
		return "json"
	}
	return "text"
}

func defaultLogOutput(env string) string {
	if env == "production" {
		return "both"
	}
	return "stdout"
}

func isValidPort(port string) bool {
	if port == "" {
		return false
	}
	var portNum int
	if _, err := fmt.Sscanf(port, "%d", &portNum); err != nil {
		return false
	}
	return portNum > 0 && portNum < 65536
}

// DSN returns the PostgreSQL connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s timezone=%s connect_timeout=%d",
		c.DB.Host, c.DB.Port, c.DB.User, c.DB.Password,
		c.DB.Name, c.DB.SSLMode, c.DB.Timezone,
		int(c.DB.ConnectTimeout.Seconds()),
	)
}
