package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	"github.com/redis/go-redis/v9"
)

type redisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

type VideoCache struct {
	client redisClient
}

func NewVideoCache(redisURL string) (*VideoCache, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	return &VideoCache{client: redis.NewClient(opts)}, nil
}

func (c *VideoCache) cacheKey(userID string) string {
	return fmt.Sprintf("video:list:%s", userID)
}

func (c *VideoCache) GetUserVideos(ctx context.Context, userID string) ([]*entity.Video, error) {
	data, err := c.client.Get(ctx, c.cacheKey(userID)).Bytes()
	if err != nil {
		return nil, fmt.Errorf("get from cache: %w", err)
	}

	var videos []*entity.Video
	if err := json.Unmarshal(data, &videos); err != nil {
		return nil, fmt.Errorf("unmarshal cached videos: %w", err)
	}

	return videos, nil
}

func (c *VideoCache) SetUserVideos(ctx context.Context, userID string, videos []*entity.Video, ttl time.Duration) error {
	data, err := json.Marshal(videos)
	if err != nil {
		return fmt.Errorf("marshal videos for cache: %w", err)
	}

	if err := c.client.Set(ctx, c.cacheKey(userID), data, ttl).Err(); err != nil {
		return fmt.Errorf("set cache: %w", err)
	}

	return nil
}

func (c *VideoCache) InvalidateUserVideos(ctx context.Context, userID string) error {
	if err := c.client.Del(ctx, c.cacheKey(userID)).Err(); err != nil {
		return fmt.Errorf("invalidate cache: %w", err)
	}
	return nil
}
