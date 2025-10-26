package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config stores all application configuration
type Config struct {
	Server  ServerConfig
	MongoDB MongoDBConfig
	Redis   RedisConfig
	Kafka   KafkaConfig
	Logging LoggingConfig
	App     AppConfig
}

// ServerConfig defines the HTTP server configuration
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Environment  string
}

// MongoDBConfig defines the MongoDB connection configuration
type MongoDBConfig struct {
	URI               string
	Database          string
	ConnectionTimeout time.Duration
	MaxPoolSize       uint64
}

// RedisConfig defines the Redis cache configuration
type RedisConfig struct {
	URL        string
	Password   string
	DB         int
	PoolSize   int
	DefaultTTL time.Duration
}

// KafkaConfig defines the Kafka configuration for producers and consumers
type KafkaConfig struct {
	Brokers        []string
	TopicOrders    string
	ConsumerGroup  string
	EnableProducer bool
}

// LoggingConfig defines logging level and format
type LoggingConfig struct {
	Level  string
	Format string
}

// AppConfig defines general application settings
type AppConfig struct {
	RequestTimeout   time.Duration
	MaxItemsPerOrder int
	DefaultPageSize  int
	MaxPageSize      int
}

// Load loads configuration from environment variables and .env file
func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	// Attempt to load .env file (optional)
	_ = viper.ReadInConfig()

	setDefaults()

	config := &Config{
		Server: ServerConfig{
			Port:         viper.GetString("PORT"),
			ReadTimeout:  viper.GetDuration("SERVER_READ_TIMEOUT"),
			WriteTimeout: viper.GetDuration("SERVER_WRITE_TIMEOUT"),
			Environment:  viper.GetString("ENV"),
		},
		MongoDB: MongoDBConfig{
			URI:               viper.GetString("MONGODB_URI"),
			Database:          viper.GetString("MONGODB_DATABASE"),
			ConnectionTimeout: viper.GetDuration("MONGODB_CONNECTION_TIMEOUT"),
			MaxPoolSize:       viper.GetUint64("MONGODB_MAX_POOL_SIZE"),
		},
		Redis: RedisConfig{
			URL:        viper.GetString("REDIS_URL"),
			Password:   viper.GetString("REDIS_PASSWORD"),
			DB:         viper.GetInt("REDIS_DB"),
			PoolSize:   viper.GetInt("REDIS_POOL_SIZE"),
			DefaultTTL: viper.GetDuration("REDIS_DEFAULT_TTL"),
		},
		Kafka: KafkaConfig{
			Brokers:        viper.GetStringSlice("KAFKA_BROKERS"),
			TopicOrders:    viper.GetString("KAFKA_TOPIC_ORDERS"),
			ConsumerGroup:  viper.GetString("KAFKA_CONSUMER_GROUP"),
			EnableProducer: viper.GetBool("KAFKA_ENABLE_PRODUCER"),
		},
		Logging: LoggingConfig{
			Level:  viper.GetString("LOG_LEVEL"),
			Format: viper.GetString("LOG_FORMAT"),
		},
		App: AppConfig{
			RequestTimeout:   viper.GetDuration("REQUEST_TIMEOUT"),
			MaxItemsPerOrder: viper.GetInt("MAX_ITEMS_PER_ORDER"),
			DefaultPageSize:  viper.GetInt("DEFAULT_PAGE_SIZE"),
			MaxPageSize:      viper.GetInt("MAX_PAGE_SIZE"),
		},
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate checks required configuration values
func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("PORT is required")
	}
	if c.MongoDB.URI == "" {
		return fmt.Errorf("MONGODB_URI is required")
	}
	if c.Redis.URL == "" {
		return fmt.Errorf("REDIS_URL is required")
	}
	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("KAFKA_BROKERS is required")
	}
	return nil
}

// setDefaults sets default values for all configuration keys
func setDefaults() {
	// Server defaults
	viper.SetDefault("ENV", "development")
	viper.SetDefault("PORT", "3000")
	viper.SetDefault("SERVER_READ_TIMEOUT", "10s")
	viper.SetDefault("SERVER_WRITE_TIMEOUT", "10s")

	// MongoDB defaults
	viper.SetDefault("MONGODB_DATABASE", "orders_db")
	viper.SetDefault("MONGODB_CONNECTION_TIMEOUT", "10s")
	viper.SetDefault("MONGODB_MAX_POOL_SIZE", 100)

	// Redis defaults
	viper.SetDefault("REDIS_DB", 0)
	viper.SetDefault("REDIS_POOL_SIZE", 10)
	viper.SetDefault("REDIS_DEFAULT_TTL", "60s")

	// Kafka defaults
	viper.SetDefault("KAFKA_TOPIC_ORDERS", "orders.events")
	viper.SetDefault("KAFKA_CONSUMER_GROUP", "orders-service")
	viper.SetDefault("KAFKA_ENABLE_PRODUCER", true)

	// Logging defaults
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_FORMAT", "json")

	// App defaults
	viper.SetDefault("REQUEST_TIMEOUT", "30s")
	viper.SetDefault("MAX_ITEMS_PER_ORDER", 100)
	viper.SetDefault("DEFAULT_PAGE_SIZE", 10)
	viper.SetDefault("MAX_PAGE_SIZE", 100)
}
