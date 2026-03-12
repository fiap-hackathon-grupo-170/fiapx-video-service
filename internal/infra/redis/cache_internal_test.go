package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	goredis "github.com/redis/go-redis/v9"
)

type mockRedis struct {
	getVal string
	getErr error
	setErr error
	delErr error
}

func (m *mockRedis) Get(_ context.Context, _ string) *goredis.StringCmd {
	cmd := goredis.NewStringCmd(context.Background())
	if m.getErr != nil {
		cmd.SetErr(m.getErr)
	} else {
		cmd.SetVal(m.getVal)
	}
	return cmd
}

func (m *mockRedis) Set(_ context.Context, _ string, _ interface{}, _ time.Duration) *goredis.StatusCmd {
	cmd := goredis.NewStatusCmd(context.Background())
	if m.setErr != nil {
		cmd.SetErr(m.setErr)
	}
	return cmd
}

func (m *mockRedis) Del(_ context.Context, _ ...string) *goredis.IntCmd {
	cmd := goredis.NewIntCmd(context.Background())
	if m.delErr != nil {
		cmd.SetErr(m.delErr)
	}
	return cmd
}

func newMockCache(mr *mockRedis) *VideoCache {
	return &VideoCache{client: mr}
}

func TestCacheKey(t *testing.T) {
	c := newMockCache(&mockRedis{})
	key := c.cacheKey("user-1")
	if key != "video:list:user-1" {
		t.Errorf("expected video:list:user-1, got %s", key)
	}
}

func TestGetUserVideos_Success(t *testing.T) {
	videos := []*entity.Video{entity.NewVideo("u1", "u@t.com", "k.mp4", "a.mp4", 100)}
	data, _ := json.Marshal(videos)

	c := newMockCache(&mockRedis{getVal: string(data)})
	result, err := c.GetUserVideos(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 video, got %d", len(result))
	}
}

func TestGetUserVideos_CacheMiss(t *testing.T) {
	c := newMockCache(&mockRedis{getErr: fmt.Errorf("redis: nil")})
	_, err := c.GetUserVideos(context.Background(), "u1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetUserVideos_InvalidJSON(t *testing.T) {
	c := newMockCache(&mockRedis{getVal: "not json"})
	_, err := c.GetUserVideos(context.Background(), "u1")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSetUserVideos_Success(t *testing.T) {
	c := newMockCache(&mockRedis{})
	videos := []*entity.Video{entity.NewVideo("u1", "u@t.com", "k.mp4", "a.mp4", 100)}
	err := c.SetUserVideos(context.Background(), "u1", videos, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetUserVideos_Error(t *testing.T) {
	c := newMockCache(&mockRedis{setErr: fmt.Errorf("redis error")})
	videos := []*entity.Video{entity.NewVideo("u1", "u@t.com", "k.mp4", "a.mp4", 100)}
	err := c.SetUserVideos(context.Background(), "u1", videos, 5*time.Minute)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInvalidateUserVideos_Success(t *testing.T) {
	c := newMockCache(&mockRedis{})
	err := c.InvalidateUserVideos(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInvalidateUserVideos_Error(t *testing.T) {
	c := newMockCache(&mockRedis{delErr: fmt.Errorf("redis error")})
	err := c.InvalidateUserVideos(context.Background(), "u1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewVideoCache_ValidURL(t *testing.T) {
	c, err := NewVideoCache("redis://localhost:6379/0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil cache")
	}
}

func TestNewVideoCache_InvalidURL(t *testing.T) {
	_, err := NewVideoCache("://invalid")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}
