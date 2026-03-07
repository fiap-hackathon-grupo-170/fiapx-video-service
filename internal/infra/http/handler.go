package http

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	"github.com/fiapx/fiapx-video-service/internal/domain/port"
	"github.com/fiapx/fiapx-video-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var validVideoExtensions = map[string]bool{
	".mp4":  true,
	".avi":  true,
	".mov":  true,
	".mkv":  true,
	".wmv":  true,
	".flv":  true,
	".webm": true,
}

type Handler struct {
	upload   *usecase.UploadVideoUseCase
	list     *usecase.ListVideosUseCase
	get      *usecase.GetVideoUseCase
	download *usecase.DownloadVideoUseCase
	delete   *usecase.DeleteVideoUseCase
	storage  port.VideoStorage
	logger   *zap.Logger
	maxBytes int64
}

func NewHandler(
	upload *usecase.UploadVideoUseCase,
	list *usecase.ListVideosUseCase,
	get *usecase.GetVideoUseCase,
	download *usecase.DownloadVideoUseCase,
	deleteUC *usecase.DeleteVideoUseCase,
	storage port.VideoStorage,
	logger *zap.Logger,
	maxUploadMB int64,
) *Handler {
	return &Handler{
		upload:   upload,
		list:     list,
		get:      get,
		download: download,
		delete:   deleteUC,
		storage:  storage,
		logger:   logger,
		maxBytes: maxUploadMB * 1024 * 1024,
	}
}

func (h *Handler) UploadHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	userEmail := c.GetString("user_email")

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxBytes)

	fileHeader, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read video file: " + err.Error()})
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !validVideoExtensions[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported video format. Allowed: mp4, avi, mov, mkv, wmv, flv, webm"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open uploaded file"})
		return
	}
	defer file.Close()

	video, err := h.upload.Execute(c.Request.Context(), file, fileHeader.Filename, fileHeader.Size, userID, userEmail)
	if err != nil {
		h.logger.Error("upload failed", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process upload"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         video.ID,
		"job_id":     video.ID,
		"status":     video.Status,
		"video_key":  video.VideoKey,
		"file_size":  video.FileSize,
		"created_at": video.CreatedAt,
	})
}

func (h *Handler) ListHandler(c *gin.Context) {
	userID := c.GetString("user_id")

	videos, err := h.list.Execute(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("list videos failed", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list videos"})
		return
	}

	if videos == nil {
		videos = []*entity.Video{}
	}

	c.JSON(http.StatusOK, gin.H{
		"videos": videos,
		"total":  len(videos),
	})
}

func (h *Handler) GetHandler(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid video id"})
		return
	}

	userID := c.GetString("user_id")

	video, err := h.get.Execute(c.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, usecase.ErrVideoNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "video not found"})
			return
		}
		h.logger.Error("get video failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get video"})
		return
	}

	c.JSON(http.StatusOK, video)
}

func (h *Handler) DownloadHandler(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid video id"})
		return
	}

	userID := c.GetString("user_id")

	result, err := h.download.Execute(c.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, usecase.ErrVideoNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "video not found"})
			return
		}
		if errors.Is(err, usecase.ErrVideoNotReady) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, usecase.ErrVideoFailed) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("download failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate download url"})
		return
	}

	stream, size, err := h.storage.StreamZip(c.Request.Context(), result.ZipKey)
	if err != nil {
		h.logger.Error("stream zip failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stream zip file"})
		return
	}
	defer stream.Close()

	filename := fmt.Sprintf("frames_%s.zip", id.String())
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Type", "application/zip")
	if size > 0 {
		c.Header("Content-Length", fmt.Sprintf("%d", size))
	}
	c.Status(http.StatusOK)
	io.Copy(c.Writer, stream) //nolint:errcheck
}

func (h *Handler) DeleteHandler(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid video id"})
		return
	}

	userID := c.GetString("user_id")

	if err := h.delete.Execute(c.Request.Context(), id, userID); err != nil {
		if errors.Is(err, usecase.ErrVideoNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "video not found"})
			return
		}
		h.logger.Error("delete failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete video"})
		return
	}

	c.Status(http.StatusNoContent)
}
