package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fiapx/fiapx-video-service/internal/domain/port"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	engine *gin.Engine
	srv    *http.Server
	logger *zap.Logger
}

func NewServer(handler *Handler, validator port.TokenValidator, port int, logger *zap.Logger) *Server {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	// CORS
	engine.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Content-Type", "application/json")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Set max multipart memory to 32MB; the rest goes to temp disk
	engine.MaxMultipartMemory = 32 << 20

	// Public routes
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Protected routes
	auth := AuthMiddleware(validator)
	api := engine.Group("/api/videos")
	api.Use(auth)
	{
		api.POST("/upload", handler.UploadHandler)
		api.GET("", handler.ListHandler)
		api.GET("/:id", handler.GetHandler)
		api.GET("/:id/download", handler.DownloadHandler)
		api.DELETE("/:id", handler.DeleteHandler)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: engine,
	}

	return &Server{engine: engine, srv: srv, logger: logger}
}

// Engine returns the underlying gin engine, useful for testing with httptest.
func (s *Server) Engine() http.Handler {
	return s.engine
}

func (s *Server) Start() error {
	s.logger.Info("HTTP server starting", zap.String("addr", s.srv.Addr))
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server error: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
