package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bogru/go-url-shortener/internal/shortener"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(db.Close)

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}

	store := &Store{db: db}
	if err := store.migrate(ctx); err != nil {
		t.Fatalf("migrate() error = %v", err)
	}

	if _, err := db.Exec(ctx, `TRUNCATE short_links`); err != nil {
		t.Fatalf("TRUNCATE error = %v", err)
	}

	return store
}

func TestStoreSaveAndGet(t *testing.T) {
	store := newTestStore(t)

	if err := store.Save("abc123_XYZ", "https://example.com"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Get("abc123_XYZ")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got != "https://example.com" {
		t.Fatalf("Get() = %q, want %q", got, "https://example.com")
	}
}

func TestStoreRejectsDuplicateCode(t *testing.T) {
	store := newTestStore(t)

	if err := store.Save("abc123_XYZ", "https://example.com/first"); err != nil {
		t.Fatalf("Save() first error = %v", err)
	}

	err := store.Save("abc123_XYZ", "https://example.com/second")
	if !errors.Is(err, shortener.ErrShortCodeBusy) {
		t.Fatalf("Save() duplicate error = %v, want %v", err, shortener.ErrShortCodeBusy)
	}
}
