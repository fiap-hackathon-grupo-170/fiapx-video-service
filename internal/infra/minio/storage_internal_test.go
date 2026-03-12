package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"testing"
	"time"

	miniogo "github.com/minio/minio-go/v7"
)

type mockMinioClient struct {
	bucketExistsResult bool
	bucketExistsErr    error
	makeBucketErr      error
	putObjectErr       error
	removeObjectErr    error
	presignedURL       *url.URL
	presignedErr       error
	getObjectErr       error
}

func (m *mockMinioClient) BucketExists(_ context.Context, _ string) (bool, error) {
	return m.bucketExistsResult, m.bucketExistsErr
}

func (m *mockMinioClient) MakeBucket(_ context.Context, _ string, _ miniogo.MakeBucketOptions) error {
	return m.makeBucketErr
}

func (m *mockMinioClient) PutObject(_ context.Context, _, _ string, _ io.Reader, _ int64, _ miniogo.PutObjectOptions) (miniogo.UploadInfo, error) {
	return miniogo.UploadInfo{}, m.putObjectErr
}

func (m *mockMinioClient) RemoveObject(_ context.Context, _, _ string, _ miniogo.RemoveObjectOptions) error {
	return m.removeObjectErr
}

func (m *mockMinioClient) PresignedGetObject(_ context.Context, _, _ string, _ time.Duration, _ url.Values) (*url.URL, error) {
	return m.presignedURL, m.presignedErr
}

func (m *mockMinioClient) GetObject(_ context.Context, _, _ string, _ miniogo.GetObjectOptions) (*miniogo.Object, error) {
	return nil, m.getObjectErr
}

func newMockStorage(mc *mockMinioClient) *Storage {
	return &Storage{
		client:       mc,
		uploadBucket: "uploads",
		zipBucket:    "zips",
	}
}

func TestEnsureBuckets_AllExist(t *testing.T) {
	s := newMockStorage(&mockMinioClient{bucketExistsResult: true})
	if err := s.EnsureBuckets(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureBuckets_CreateNew(t *testing.T) {
	s := newMockStorage(&mockMinioClient{bucketExistsResult: false})
	if err := s.EnsureBuckets(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureBuckets_ExistsError(t *testing.T) {
	s := newMockStorage(&mockMinioClient{bucketExistsErr: fmt.Errorf("conn failed")})
	if err := s.EnsureBuckets(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestEnsureBuckets_MakeError(t *testing.T) {
	s := newMockStorage(&mockMinioClient{bucketExistsResult: false, makeBucketErr: fmt.Errorf("denied")})
	if err := s.EnsureBuckets(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestUploadVideo_Success(t *testing.T) {
	s := newMockStorage(&mockMinioClient{})
	err := s.UploadVideo(context.Background(), "key.mp4", bytes.NewReader([]byte("data")), 4, "video/mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadVideo_Error(t *testing.T) {
	s := newMockStorage(&mockMinioClient{putObjectErr: fmt.Errorf("upload failed")})
	err := s.UploadVideo(context.Background(), "key.mp4", bytes.NewReader([]byte("data")), 4, "video/mp4")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteVideo_Success(t *testing.T) {
	s := newMockStorage(&mockMinioClient{})
	if err := s.DeleteVideo(context.Background(), "key.mp4"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteVideo_Error(t *testing.T) {
	s := newMockStorage(&mockMinioClient{removeObjectErr: fmt.Errorf("remove failed")})
	if err := s.DeleteVideo(context.Background(), "key.mp4"); err == nil {
		t.Fatal("expected error")
	}
}

func TestPresignedDownloadURL_Success(t *testing.T) {
	u, _ := url.Parse("http://minio:9000/zips/test.zip?token=abc")
	s := newMockStorage(&mockMinioClient{presignedURL: u})

	result, err := s.PresignedDownloadURL(context.Background(), "test.zip", time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != u.String() {
		t.Errorf("expected %s, got %s", u.String(), result)
	}
}

func TestPresignedDownloadURL_WithPublicURL(t *testing.T) {
	u, _ := url.Parse("http://minio:9000/zips/test.zip?token=abc")
	s := newMockStorage(&mockMinioClient{presignedURL: u})
	s.publicURL = "http://localhost:9000"

	result, err := s.PresignedDownloadURL(context.Background(), "test.zip", time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "http://localhost:9000/zips/test.zip?token=abc"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestPresignedDownloadURL_Error(t *testing.T) {
	s := newMockStorage(&mockMinioClient{presignedErr: fmt.Errorf("presign failed")})
	_, err := s.PresignedDownloadURL(context.Background(), "test.zip", time.Hour)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStreamZip_GetObjectError(t *testing.T) {
	s := newMockStorage(&mockMinioClient{getObjectErr: fmt.Errorf("not found")})
	_, _, err := s.StreamZip(context.Background(), "test.zip")
	if err == nil {
		t.Fatal("expected error")
	}
}
