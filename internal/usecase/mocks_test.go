package usecase_test

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	"github.com/google/uuid"
)

// ---- Repository mock ----

type mockRepo struct {
	videos    map[uuid.UUID]*entity.Video
	createErr error
	updateErr error
	findErr   error
	deleteErr error
}

func newMockRepo() *mockRepo {
	return &mockRepo{videos: make(map[uuid.UUID]*entity.Video)}
}

func (m *mockRepo) Create(_ context.Context, v *entity.Video) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.videos[v.ID] = v
	return nil
}

func (m *mockRepo) Update(_ context.Context, v *entity.Video) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.videos[v.ID] = v
	return nil
}

func (m *mockRepo) FindByID(_ context.Context, id uuid.UUID) (*entity.Video, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	v, ok := m.videos[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return v, nil
}

func (m *mockRepo) FindByUserID(_ context.Context, userID string) ([]*entity.Video, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	var result []*entity.Video
	for _, v := range m.videos {
		if v.UserID == userID {
			result = append(result, v)
		}
	}
	return result, nil
}

func (m *mockRepo) Delete(_ context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.videos, id)
	return nil
}

// ---- Storage mock ----

type mockStorage struct {
	uploadErr    error
	deleteErr    error
	presignURL   string
	presignErr   error
	streamReader io.ReadCloser
	streamSize   int64
	streamErr    error
}

func (m *mockStorage) UploadVideo(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error {
	return m.uploadErr
}

func (m *mockStorage) DeleteVideo(_ context.Context, _ string) error {
	return m.deleteErr
}

func (m *mockStorage) PresignedDownloadURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	if m.presignErr != nil {
		return "", m.presignErr
	}
	return m.presignURL, nil
}

func (m *mockStorage) StreamZip(_ context.Context, _ string) (io.ReadCloser, int64, error) {
	if m.streamErr != nil {
		return nil, 0, m.streamErr
	}
	return m.streamReader, m.streamSize, nil
}

// ---- Publisher mock ----

type mockPublisher struct {
	publishErr    error
	publishedMsgs []entity.VideoUploadedMessage
}

func (m *mockPublisher) PublishVideoUploaded(_ context.Context, msg entity.VideoUploadedMessage) error {
	if m.publishErr != nil {
		return m.publishErr
	}
	m.publishedMsgs = append(m.publishedMsgs, msg)
	return nil
}

// ---- Cache mock ----

type mockCache struct {
	store          map[string][]*entity.Video
	getErr         error
	setErr         error
	invalidateErr  error
	invalidateCalls []string
}

func newMockCache() *mockCache {
	return &mockCache{store: make(map[string][]*entity.Video)}
}

func (m *mockCache) GetUserVideos(_ context.Context, userID string) ([]*entity.Video, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	v, ok := m.store[userID]
	if !ok {
		return nil, errors.New("cache miss")
	}
	return v, nil
}

func (m *mockCache) SetUserVideos(_ context.Context, userID string, videos []*entity.Video, _ time.Duration) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.store[userID] = videos
	return nil
}

func (m *mockCache) InvalidateUserVideos(_ context.Context, userID string) error {
	m.invalidateCalls = append(m.invalidateCalls, userID)
	return m.invalidateErr
}
