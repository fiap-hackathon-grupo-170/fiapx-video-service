package postgres

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// --- Mock pool ---

type mockPool struct {
	execErr     error
	queryRowFn  func(ctx context.Context, sql string, args ...any) pgx.Row
	queryFn     func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (m *mockPool) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, m.execErr
}

func (m *mockPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.queryRowFn != nil {
		return m.queryRowFn(ctx, sql, args...)
	}
	return &mockRow{scanErr: fmt.Errorf("not configured")}
}

func (m *mockPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, sql, args...)
	}
	return nil, fmt.Errorf("not configured")
}

// --- Mock Row ---

type mockRow struct {
	data    []any
	scanErr error
}

func (r *mockRow) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	for i := range dest {
		dv := reflect.ValueOf(dest[i]).Elem()
		sv := reflect.ValueOf(r.data[i])
		if r.data[i] == nil {
			dv.Set(reflect.Zero(dv.Type()))
		} else {
			dv.Set(sv)
		}
	}
	return nil
}

// --- Mock Rows ---

type mockRows struct {
	data    [][]any
	idx     int
	scanErr error
	errVal  error
}

func (r *mockRows) Close()                                        {}
func (r *mockRows) Err() error                                     { return r.errVal }
func (r *mockRows) CommandTag() pgconn.CommandTag                  { return pgconn.CommandTag{} }
func (r *mockRows) FieldDescriptions() []pgconn.FieldDescription   { return nil }
func (r *mockRows) Next() bool                                     { r.idx++; return r.idx <= len(r.data) }
func (r *mockRows) Values() ([]any, error)                         { return nil, nil }
func (r *mockRows) RawValues() [][]byte                            { return nil }
func (r *mockRows) Conn() *pgx.Conn                               { return nil }

func (r *mockRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	row := r.data[r.idx-1]
	for i := range dest {
		dv := reflect.ValueOf(dest[i]).Elem()
		sv := reflect.ValueOf(row[i])
		if row[i] == nil {
			dv.Set(reflect.Zero(dv.Type()))
		} else {
			dv.Set(sv)
		}
	}
	return nil
}

// --- Tests ---

func TestNewVideoRepository(t *testing.T) {
	repo := NewVideoRepository(nil)
	if repo == nil {
		t.Fatal("expected non-nil repo")
	}
}

func TestCreate_Success(t *testing.T) {
	mp := &mockPool{}
	repo := &VideoRepository{pool: mp}
	video := entity.NewVideo("user-1", "u@test.com", "key.mp4", "test.mp4", 1024)

	err := repo.Create(context.Background(), video)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreate_Error(t *testing.T) {
	mp := &mockPool{execErr: fmt.Errorf("db error")}
	repo := &VideoRepository{pool: mp}
	video := entity.NewVideo("user-1", "u@test.com", "key.mp4", "test.mp4", 1024)

	err := repo.Create(context.Background(), video)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdate_Success(t *testing.T) {
	mp := &mockPool{}
	repo := &VideoRepository{pool: mp}
	video := entity.NewVideo("user-1", "u@test.com", "key.mp4", "test.mp4", 1024)

	err := repo.Update(context.Background(), video)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdate_Error(t *testing.T) {
	mp := &mockPool{execErr: fmt.Errorf("db error")}
	repo := &VideoRepository{pool: mp}
	video := entity.NewVideo("user-1", "u@test.com", "key.mp4", "test.mp4", 1024)

	err := repo.Update(context.Background(), video)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFindByID_Success(t *testing.T) {
	id := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	var completedAt *time.Time

	mp := &mockPool{
		queryRowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return &mockRow{data: []any{
				id, "user-1", "u@test.com", "key.mp4", "zip.zip",
				"test.mp4", "PENDING",
				0, int64(1024), float64(0), "",
				now, now, completedAt,
			}}
		},
	}
	repo := &VideoRepository{pool: mp}

	video, err := repo.FindByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if video.ID != id {
		t.Errorf("expected id %s, got %s", id, video.ID)
	}
	if video.Status != entity.VideoStatusPending {
		t.Errorf("expected PENDING, got %s", video.Status)
	}
}

func TestFindByID_Error(t *testing.T) {
	mp := &mockPool{
		queryRowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return &mockRow{scanErr: fmt.Errorf("not found")}
		},
	}
	repo := &VideoRepository{pool: mp}

	_, err := repo.FindByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFindByUserID_Success(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	var completedAt *time.Time

	mp := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return &mockRows{data: [][]any{
				{id1, "user-1", "u@test.com", "k1.mp4", "z1.zip", "a.mp4", "PENDING", 0, int64(100), float64(0), "", now, now, completedAt},
				{id2, "user-1", "u@test.com", "k2.mp4", "z2.zip", "b.mp4", "COMPLETED", 10, int64(200), float64(5.0), "", now, now, completedAt},
			}}, nil
		},
	}
	repo := &VideoRepository{pool: mp}

	videos, err := repo.FindByUserID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(videos) != 2 {
		t.Fatalf("expected 2 videos, got %d", len(videos))
	}
}

func TestFindByUserID_QueryError(t *testing.T) {
	mp := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	repo := &VideoRepository{pool: mp}

	_, err := repo.FindByUserID(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFindByUserID_ScanError(t *testing.T) {
	mp := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return &mockRows{data: [][]any{{}}, scanErr: fmt.Errorf("scan error")}, nil
		},
	}
	repo := &VideoRepository{pool: mp}

	_, err := repo.FindByUserID(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFindByUserID_RowsError(t *testing.T) {
	mp := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return &mockRows{data: [][]any{}, errVal: fmt.Errorf("rows error")}, nil
		},
	}
	repo := &VideoRepository{pool: mp}

	_, err := repo.FindByUserID(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDelete_Success(t *testing.T) {
	mp := &mockPool{}
	repo := &VideoRepository{pool: mp}

	err := repo.Delete(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_Error(t *testing.T) {
	mp := &mockPool{execErr: fmt.Errorf("db error")}
	repo := &VideoRepository{pool: mp}

	err := repo.Delete(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error")
	}
}
