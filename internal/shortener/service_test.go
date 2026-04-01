package shortener

import (
	"errors"
	"testing"
)

type mockStore struct {
	data map[string]string
}

func newMockStore() *mockStore {
	return &mockStore{data: make(map[string]string)}
}

func (m *mockStore) Get(shortCode string) (string, error) {
	url, ok := m.data[shortCode]
	if !ok {
		return "", ErrShortCodeMiss
	}
	return url, nil
}

func (m *mockStore) Save(shortCode, originalURL string) error {
	if _, exists := m.data[shortCode]; exists {
		return ErrShortCodeBusy
	}
	m.data[shortCode] = originalURL
	return nil
}

func TestServiceShorten_HappyPath(t *testing.T) {
	svc := NewService(newMockStore())

	result, err := svc.Shorten("https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ShortCode == "" {
		t.Fatal("expected non-empty short code")
	}
	if result.AlreadyExists {
		t.Fatal("expected AlreadyExists=false on first save")
	}
	if result.OriginalURL != "https://example.com" {
		t.Fatalf("expected OriginalURL %q, got %q", "https://example.com", result.OriginalURL)
	}
}

func TestServiceShorten_InvalidURL(t *testing.T) {
	svc := NewService(newMockStore())

	_, err := svc.Shorten("not-a-url")
	if !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("expected ErrInvalidURL, got %v", err)
	}
}

func TestServiceShorten_AlreadyExists(t *testing.T) {
	svc := NewService(newMockStore())
	const url = "https://example.com"

	first, err := svc.Shorten(url)
	if err != nil {
		t.Fatalf("first Shorten() error: %v", err)
	}

	second, err := svc.Shorten(url)
	if err != nil {
		t.Fatalf("second Shorten() error: %v", err)
	}

	if !second.AlreadyExists {
		t.Fatal("expected AlreadyExists=true on second call")
	}
	if first.ShortCode != second.ShortCode {
		t.Fatalf("expected same short code, got %q and %q", first.ShortCode, second.ShortCode)
	}
}

func TestServiceShorten_ConcurrentSameURL(t *testing.T) {
	store := &busyThenMatchStore{url: "https://example.com", code: buildShortCode("https://example.com", 0)}
	svc := NewService(store)

	result, err := svc.Shorten("https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.AlreadyExists {
		t.Fatal("expected AlreadyExists=true when another goroutine won the race with the same URL")
	}
}

type busyThenMatchStore struct {
	url  string
	code string
}

func (b *busyThenMatchStore) Get(shortCode string) (string, error) {
	if shortCode == b.code {
		return b.url, nil
	}
	return "", ErrShortCodeMiss
}

func (b *busyThenMatchStore) Save(shortCode, _ string) error {
	if shortCode == b.code {
		return ErrShortCodeBusy
	}
	return nil
}

func TestServiceShorten_SaltCollision(t *testing.T) {
	store := newMockStore()
	const url = "https://example.com"

	// Occupy the slot that salt=0 would produce with a different URL.
	code0 := buildShortCode(url, 0)
	store.data[code0] = "https://other.com"

	svc := NewService(store)
	result, err := svc.Shorten(url)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := buildShortCode(url, 1)
	if result.ShortCode != expected {
		t.Fatalf("expected code from salt=1 %q, got %q", expected, result.ShortCode)
	}
}

func TestServiceResolve_NotFound(t *testing.T) {
	svc := NewService(newMockStore())

	_, err := svc.Resolve("nonexistent")
	if !errors.Is(err, ErrShortCodeMiss) {
		t.Fatalf("expected ErrShortCodeMiss, got %v", err)
	}
}

func TestBuildShortCode_Deterministic(t *testing.T) {
	code1 := buildShortCode("https://example.com", 0)
	code2 := buildShortCode("https://example.com", 0)
	if code1 != code2 {
		t.Fatalf("expected deterministic output, got %q and %q", code1, code2)
	}
}

func TestBuildShortCode_Length(t *testing.T) {
	code := buildShortCode("https://example.com", 0)
	if len(code) != shortCodeLength {
		t.Fatalf("expected length %d, got %d", shortCodeLength, len(code))
	}
}

func TestBuildShortCode_DifferentSaltGivesDifferentCode(t *testing.T) {
	code0 := buildShortCode("https://example.com", 0)
	code1 := buildShortCode("https://example.com", 1)
	if code0 == code1 {
		t.Fatalf("expected different codes for different salts, both got %q", code0)
	}
}

func TestBuildShortCode_AlphabetOnly(t *testing.T) {
	code := buildShortCode("https://example.com", 0)
	for _, ch := range code {
		if !isInAlphabet(byte(ch)) {
			t.Fatalf("unexpected character %q in short code %q", ch, code)
		}
	}
}

func isInAlphabet(b byte) bool {
	for i := 0; i < len(alphabet); i++ {
		if alphabet[i] == b {
			return true
		}
	}
	return false
}
