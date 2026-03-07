package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fiapx/fiapx-video-service/internal/infra/config"
	httpinfra "github.com/fiapx/fiapx-video-service/internal/infra/http"
	"github.com/fiapx/fiapx-video-service/internal/infra/keycloak"
	"github.com/fiapx/fiapx-video-service/internal/infra/metrics"
	minioinfra "github.com/fiapx/fiapx-video-service/internal/infra/minio"
	"github.com/fiapx/fiapx-video-service/internal/infra/postgres"
	"github.com/fiapx/fiapx-video-service/internal/infra/rabbitmq"
	redisinfra "github.com/fiapx/fiapx-video-service/internal/infra/redis"
	"github.com/fiapx/fiapx-video-service/internal/infra/tracing"
	"github.com/fiapx/fiapx-video-service/internal/usecase"
	"github.com/fiapx/fiapx-video-service/pkg/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	fatalOnErr(err, "load config")

	log, err := logger.New(cfg.LogLevel)
	fatalOnErr(err, "init logger")
	defer log.Sync()

	log.Info("starting fiapx-video-service")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Tracing (non-fatal if Jaeger unavailable)
	tp, err := tracing.InitTracer(ctx, cfg.JaegerEndpoint)
	if err != nil {
		log.Warn("tracing init failed, continuing without tracing", zap.Error(err))
	} else {
		defer tp.Shutdown(ctx)
	}

	// Database
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	fatalOnErr(err, "connect to postgres")
	defer pool.Close()

	// Migrations
	if err := postgres.RunMigrations(cfg.DatabaseURL, "migrations"); err != nil {
		log.Warn("migration warning", zap.Error(err))
	}

	// MinIO
	storage, err := minioinfra.NewStorage(minioinfra.StorageConfig{
		Endpoint:     cfg.MinIOEndpoint,
		AccessKey:    cfg.MinIOAccessKey,
		SecretKey:    cfg.MinIOSecretKey,
		UseSSL:       cfg.MinIOUseSSL,
		UploadBucket: cfg.MinIOUploadBucket,
		ZipBucket:    cfg.MinIOZipBucket,
	})
	fatalOnErr(err, "create minio storage")
	fatalOnErr(storage.EnsureBuckets(ctx), "ensure minio buckets")

	// RabbitMQ connection (shared for publisher and consumer)
	rmqConn, err := amqp.Dial(cfg.RabbitMQURL)
	fatalOnErr(err, "connect to rabbitmq")
	defer rmqConn.Close()

	publisher, err := rabbitmq.NewPublisher(rmqConn, cfg.RabbitMQExchange)
	fatalOnErr(err, "create rabbitmq publisher")

	// Redis
	cache, err := redisinfra.NewVideoCache(cfg.RedisURL)
	fatalOnErr(err, "connect to redis")

	// Keycloak token validator
	tokenValidator := keycloak.NewTokenValidator(cfg.KeycloakURL, cfg.KeycloakRealm, log)

	// Repository
	repo := postgres.NewVideoRepository(pool)

	// Use cases
	uploadUC := usecase.NewUploadVideoUseCase(repo, storage, publisher, cache, log)
	listUC := usecase.NewListVideosUseCase(repo, cache, log)
	getUC := usecase.NewGetVideoUseCase(repo, log)
	downloadUC := usecase.NewDownloadVideoUseCase(repo, storage, log)
	deleteUC := usecase.NewDeleteVideoUseCase(repo, storage, cache, log)
	statusUC := usecase.NewHandleStatusUpdateUseCase(repo, cache, log)

	// Metrics server
	metricsSrv := metrics.StartMetricsServer(ctx, cfg.MetricsPort, log)

	// Status consumer
	statusConsumer := rabbitmq.NewStatusConsumer(rmqConn, cfg.RabbitMQStatusQ, statusUC, log)

	go func() {
		if err := statusConsumer.Start(ctx); err != nil {
			log.Error("status consumer error", zap.Error(err))
		}
	}()

	// HTTP server
	handler := httpinfra.NewHandler(uploadUC, listUC, getUC, downloadUC, deleteUC, log, cfg.MaxUploadSizeMB)
	httpServer := httpinfra.NewServer(handler, tokenValidator, cfg.HTTPPort, log)

	go func() {
		if err := httpServer.Start(); err != nil {
			log.Error("http server error", zap.Error(err))
		}
	}()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	log.Info("received shutdown signal", zap.String("signal", sig.String()))
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("http server shutdown error", zap.Error(err))
	}

	metricsSrv.Shutdown(shutdownCtx)
	statusConsumer.Close()

	log.Info("fiapx-video-service stopped")
}

func fatalOnErr(err error, msg string) {
	if err != nil {
		panic(msg + ": " + err.Error())
	}
}
