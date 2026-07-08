package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Database      DatabaseConfig
	Redis         RedisConfig
	Server        ServerConfig
	RateLimit     RateLimitConfig
	BatchProcess  BatchProcessConfig
	WorkerPool    WorkerPoolConfig
	WebSocket     WebSocketConfig
	Cache         CacheConfig
	JWT           JWTConfig
	Monitoring    MonitoringConfig
	Grafana       GrafanaConfig
	Referral      ReferralConfig
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port   string
	AppEnv string
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int
}

// BatchProcessConfig holds batch processing configuration
type BatchProcessConfig struct {
	BatchSize      int
	IntervalMS     int
	AdaptiveMode   bool
}

// WorkerPoolConfig holds worker pool configuration
type WorkerPoolConfig struct {
	WorkerCount    int
	BatchSize      int
	TickerMS       int
}

// WebSocketConfig holds WebSocket configuration
type WebSocketConfig struct {
	BufferSize        int
	HeartbeatInterval time.Duration
	WriteWait         time.Duration
	PongWait          time.Duration
}

// CacheConfig holds cache TTL configuration
type CacheConfig struct {
	ChannelTTL time.Duration
	TariffTTL  time.Duration
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret      string
	ExpiryHours int
}

// MonitoringConfig holds monitoring configuration
type MonitoringConfig struct {
	EnableMetrics bool
	MetricsPort   string
}

// GrafanaConfig holds Grafana configuration (optional feature)
type GrafanaConfig struct {
	AdminUser     string
	AdminPassword string
	RootURL       string
	Plugins       string
}

// ReferralConfig holds referral system configuration
type ReferralConfig struct {
	Enabled         bool
	CodeLength      int
	ExpiryDays      int
	MaxUsesPerCode  int
	CacheTTL        int
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if exists
	_ = godotenv.Load()
	// Load .env.local file to override with local values (should not be committed)
	_ = godotenv.Overload(".env.local")

	// Validate required environment variables first
	if err := validateRequiredEnvVars(); err != nil {
		return nil, err
	}

	cfg := &Config{
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "hudhud_admin"),
			Password:        getEnv("DB_PASSWORD", "hudhud_pass"),
			Name:            getEnv("DB_NAME", "al_hudhud"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 100),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: time.Duration(getEnvAsInt("DB_CONN_MAX_LIFETIME", 3600)) * time.Second,
			ConnMaxIdleTime: time.Duration(getEnvAsInt("DB_CONN_MAX_IDLE_TIME", 600)) * time.Second,
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Server: ServerConfig{
			Port:   getEnv("PORT", "8080"),
			AppEnv: getEnv("APP_ENV", "development"),
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: getEnvAsInt("RATE_LIMIT_REQUESTS_PER_MINUTE", 100),
		},
		BatchProcess: BatchProcessConfig{
			BatchSize:    getEnvAsInt("BATCH_SIZE", 100),
			IntervalMS:   getEnvAsInt("BATCH_INTERVAL_MS", 100),
			AdaptiveMode: getEnvAsBool("ADAPTIVE_BATCH_MODE", true),
		},
		WorkerPool: WorkerPoolConfig{
			WorkerCount: getEnvAsInt("WORKER_COUNT", 20),
			BatchSize:   getEnvAsInt("WORKER_BATCH_SIZE", 50),
			TickerMS:    getEnvAsInt("WORKER_TICKER_MS", 100),
		},
		WebSocket: WebSocketConfig{
			BufferSize:        getEnvAsInt("WS_BUFFER_SIZE", 1000),
			HeartbeatInterval: time.Duration(getEnvAsInt("WS_HEARTBEAT_INTERVAL", 30)) * time.Second,
			WriteWait:         time.Duration(getEnvAsInt("WS_WRITE_WAIT", 10)) * time.Second,
			PongWait:          time.Duration(getEnvAsInt("WS_PONG_WAIT", 60)) * time.Second,
		},
		Cache: CacheConfig{
			ChannelTTL: time.Duration(getEnvAsInt("CHANNEL_CACHE_TTL", 300)) * time.Second,
			TariffTTL:  time.Duration(getEnvAsInt("TARIFF_CACHE_TTL", 600)) * time.Second,
		},
		JWT: JWTConfig{
			Secret:      getEnv("JWT_SECRET", "change-me-in-production"),
			ExpiryHours: getEnvAsInt("JWT_EXPIRY_HOURS", 24),
		},
		Monitoring: MonitoringConfig{
			EnableMetrics: getEnvAsBool("ENABLE_METRICS", true),
			MetricsPort:   getEnv("METRICS_PORT", "9091"),
		},
		Grafana: GrafanaConfig{
			AdminUser:     getEnv("GRAFANA_ADMIN_USER", ""),
			AdminPassword: getEnv("GRAFANA_ADMIN_PASSWORD", ""),
			RootURL:       getEnv("GRAFANA_ROOT_URL", ""),
			Plugins:       getEnv("GRAFANA_PLUGINS", ""),
		},
		Referral: ReferralConfig{
			Enabled:        getEnvAsBool("REFERRAL_ENABLED", true),
			CodeLength:     getEnvAsInt("REFERRAL_CODE_LENGTH", 8),
			ExpiryDays:     getEnvAsInt("REFERRAL_EXPIRY_DAYS", 30),
			MaxUsesPerCode: getEnvAsInt("REFERRAL_MAX_USES_PER_CODE", 100),
			CacheTTL:       getEnvAsInt("REFERRAL_CACHE_TTL", 300),
		},
	}

	// Perform additional validation
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// GetDatabaseDSN returns PostgreSQL connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// GetRedisAddr returns Redis connection address
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%s", c.Redis.Host, c.Redis.Port)
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// validateRequiredEnvVars checks that all required environment variables are set
func validateRequiredEnvVars() error {
	requiredVars := []string{
		"DB_HOST",
		"DB_USER",
		"DB_PASSWORD",
		"DB_NAME",
		"REDIS_HOST",
		"JWT_SECRET",
	}

	var missing []string
	for _, key := range requiredVars {
		if os.Getenv(key) == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missing)
	}

	return nil
}

// validateRedisPassword checks if REDIS_PASSWORD is set and secure
func validateRedisPassword(password, appEnv string) error {
	// Trim whitespace for validation
	trimmedPassword := strings.TrimSpace(password)
	
	// Check for empty password
	if trimmedPassword == "" {
		return fmt.Errorf(
			"REDIS_PASSWORD is empty. Redis is running without authentication.\n"+
			"This is a security risk. Please generate a secure random password using:\n"+
			"  openssl rand -base64 32\n"+
			"Then update REDIS_PASSWORD in your .env file and redis.conf with the generated value.",
		)
	}
	
	// Check for whitespace-only passwords
	if password != trimmedPassword {
		return fmt.Errorf(
			"REDIS_PASSWORD contains only whitespace characters.\n"+
			"Please generate a secure random password using:\n"+
			"  openssl rand -base64 32\n"+
			"Then update REDIS_PASSWORD in your .env file and redis.conf with the generated value.",
		)
	}
	
	// Check minimum length (8 characters minimum for basic security)
	// Use RuneCountInString to count actual characters, not bytes
	passwordLength := utf8.RuneCountInString(trimmedPassword)
	if passwordLength < 8 {
		return fmt.Errorf(
			"REDIS_PASSWORD is too short (current length: %d, minimum: 8 characters).\n"+
			"Please generate a secure random password using:\n"+
			"  openssl rand -base64 32\n"+
			"Then update REDIS_PASSWORD in your .env file and redis.conf with the generated value.",
			passwordLength,
		)
	}
	
	return nil
}

// Validate performs additional validation checks on the configuration
func (c *Config) Validate() error {
	// Validate JWT secret strength
	if err := validateJWTSecret(c.JWT.Secret); err != nil {
		return err
	}

	// Validate Redis password in production
	if err := validateRedisPassword(c.Redis.Password, c.Server.AppEnv); err != nil {
		return err
	}

	// Validate port ranges
	if err := validatePort("DB_PORT", c.Database.Port); err != nil {
		return err
	}
	if err := validatePort("REDIS_PORT", c.Redis.Port); err != nil {
		return err
	}
	if err := validatePort("PORT", c.Server.Port); err != nil {
		return err
	}
	if err := validatePort("METRICS_PORT", c.Monitoring.MetricsPort); err != nil {
		return err
	}

	// Optional: Validate Grafana configuration (log warning if missing, don't error)
	validateGrafanaConfig(c.Grafana)

	return nil
}

// validateJWTSecret checks if JWT_SECRET is secure
func validateJWTSecret(secret string) error {
	// Check for default/placeholder values
	insecureDefaults := []string{
		"change-me-in-production",
		"change-me-in-production-to-a-secure-random-string",
		"your_secure_jwt_secret_here_minimum_32_characters",
		"secret",
		"jwt_secret",
	}
	
	for _, insecure := range insecureDefaults {
		if secret == insecure {
			return fmt.Errorf(
				"JWT_SECRET is set to an insecure default value.\n"+
				"Please generate a secure random secret using:\n"+
				"  openssl rand -base64 48\n"+
				"Then update JWT_SECRET in your .env file with the generated value.",
			)
		}
	}
	
	// Check minimum length (48 base64 chars = 36 bytes of entropy)
	if len(secret) < 48 {
		return fmt.Errorf(
			"JWT_SECRET is too short (current length: %d, minimum: 48 characters).\n"+
			"Please generate a secure random secret using:\n"+
			"  openssl rand -base64 48\n"+
			"Then update JWT_SECRET in your .env file with the generated value.",
			len(secret),
		)
	}
	
	return nil
}

// validatePort checks if a port string is valid (1-65535)
func validatePort(name, portStr string) error {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("%s must be a valid integer: %w", name, err)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("%s must be between 1 and 65535 (got: %d)", name, port)
	}
	return nil
}

// validateGrafanaConfig checks Grafana configuration and logs warnings if incomplete
// This is optional - the system will continue to work without Grafana
func validateGrafanaConfig(cfg GrafanaConfig) {
	missingVars := []string{}
	
	if cfg.AdminUser == "" {
		missingVars = append(missingVars, "GRAFANA_ADMIN_USER")
	}
	if cfg.AdminPassword == "" {
		missingVars = append(missingVars, "GRAFANA_ADMIN_PASSWORD")
	}
	if cfg.RootURL == "" {
		missingVars = append(missingVars, "GRAFANA_ROOT_URL")
	}
	
	if len(missingVars) > 0 {
		fmt.Printf("WARNING: Grafana configuration incomplete. Missing variables: %v\n", missingVars)
		fmt.Println("Grafana is an optional monitoring feature. The system will continue to work without it.")
		fmt.Println("To enable Grafana, set the following in your .env file:")
		fmt.Println("  GRAFANA_ADMIN_USER=admin")
		fmt.Println("  GRAFANA_ADMIN_PASSWORD=<secure-password>  # Generate with: openssl rand -base64 24")
		fmt.Println("  GRAFANA_ROOT_URL=http://localhost:3000")
		fmt.Println("  GRAFANA_PLUGINS=  # Optional plugins")
	}
}
