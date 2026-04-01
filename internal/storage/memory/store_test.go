package memory

import (
	"errors"
	"testing"

	"github.com/bogru/go-url-shortener/internal/shortener"
)

func TestStoreSaveAndGet(t *testing.T) {
	store := New()

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
	store := New()

	if err := store.Save("abc123_XYZ", "https://example.com/first"); err != nil {
		t.Fatalf("Save() first error = %v", err)
	}

	err := store.Save("abc123_XYZ", "https://example.com/second")
	if !errors.Is(err, shortener.ErrShortCodeBusy) {
		t.Fatalf("Save() duplicate error = %v, want %v", err, shortener.ErrShortCodeBusy)
	}
}
