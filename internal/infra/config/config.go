package config

import (
	"github.com/caarlos0/env/v11"
)

type Config struct {
	// HTTP
	HTTPPort    int `env:"HTTP_PORT"    envDefault:"8082"`
	MetricsPort int `env:"METRICS_PORT" envDefault:"8083"`

	// PostgreSQL (postgres-videos)
	DatabaseURL string `env:"DATABASE_URL" envDefault:"postgresql://video_user:video_pass@postgres-videos:5432/videos?sslmode=disable"`

	// MinIO
	MinIOEndpoint     string `env:"MINIO_ENDPOINT"      envDefault:"minio:9000"`
	MinIOAccessKey    string `env:"MINIO_ACCESS_KEY"    envDefault:"minioadmin"`
	MinIOSecretKey    string `env:"MINIO_SECRET_KEY"    envDefault:"minioadmin"`
	MinIOUseSSL       bool   `env:"MINIO_USE_SSL"       envDefault:"false"`
	MinIOUploadBucket string `env:"MINIO_UPLOAD_BUCKET" envDefault:"uploads"`
	MinIOZipBucket    string `env:"MINIO_ZIP_BUCKET"    envDefault:"zips"`
	MinIOPublicURL    string `env:"MINIO_PUBLIC_URL"    envDefault:""`

	// RabbitMQ
	RabbitMQURL         string `env:"RABBITMQ_URL"          envDefault:"amqp://guest:guest@rabbitmq:5672/"`
	RabbitMQExchange    string `env:"RABBITMQ_EXCHANGE"     envDefault:"fiapx.video"`
	RabbitMQProcessingQ string `env:"RABBITMQ_PROCESSING_Q" envDefault:"video.processing"`
	RabbitMQStatusQ     string `env:"RABBITMQ_STATUS_Q"     envDefault:"video.status"`

	// Redis
	RedisURL string `env:"REDIS_URL" envDefault:"redis://redis:6379/0"`

	// Keycloak JWT
	KeycloakURL   string `env:"KEYCLOAK_URL"   envDefault:"http://keycloak:8080"`
	KeycloakRealm string `env:"KEYCLOAK_REALM" envDefault:"fiapx"`

	// Observabilidade
	JaegerEndpoint string `env:"JAEGER_ENDPOINT" envDefault:"http://jaeger:4318/v1/traces"`
	LogLevel       string `env:"LOG_LEVEL"       envDefault:"info"`

	// Upload
	MaxUploadSizeMB int64 `env:"MAX_UPLOAD_SIZE_MB" envDefault:"500"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
