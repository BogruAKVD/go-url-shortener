package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bogru/go-url-shortener/internal/httpapi"
	"github.com/bogru/go-url-shortener/internal/shortener"
	"github.com/bogru/go-url-shortener/internal/storage/memory"
)

type shortenResp struct {
	ShortCode     string `json:"short_code"`
	ShortURL      string `json:"short_url"`
	OriginalURL   string `json:"original_url"`
	AlreadyExists bool   `json:"already_exists"`
}

type resolveResp struct {
	OriginalURL string `json:"original_url"`
}

func newTestMux() *http.ServeMux {
	svc := shortener.NewService(memory.New())
	h := httpapi.NewHandler(svc, "http://localhost:8080")
	mux := http.NewServeMux()
	h.Register(mux)
	return mux
}

func TestHandleShorten_Created(t *testing.T) {
	mux := newTestMux()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"url":"https://example.com"}`))
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var resp shortenResp
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.ShortCode == "" {
		t.Fatal("expected non-empty short_code")
	}
	if resp.AlreadyExists {
		t.Fatal("expected already_exists=false")
	}
	if resp.OriginalURL != "https://example.com" {
		t.Fatalf("expected original_url %q, got %q", "https://example.com", resp.OriginalURL)
	}
}

func TestHandleShorten_AlreadyExists(t *testing.T) {
	mux := newTestMux()
	body := `{"url":"https://example.com"}`

	w1 := httptest.NewRecorder()
	mux.ServeHTTP(w1, httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body)))

	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body)))

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 on duplicate, got %d", w2.Code)
	}

	var resp shortenResp
	if err := json.NewDecoder(w2.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !resp.AlreadyExists {
		t.Fatal("expected already_exists=true on second request")
	}

	var resp1 shortenResp
	json.NewDecoder(w1.Body).Decode(&resp1) //nolint:errcheck
	if resp.ShortCode != resp1.ShortCode {
		t.Fatalf("expected same short code, got %q and %q", resp1.ShortCode, resp.ShortCode)
	}
}

func TestHandleShorten_InvalidURL(t *testing.T) {
	mux := newTestMux()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"url":"not-a-url"}`))
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleShorten_InvalidJSON(t *testing.T) {
	mux := newTestMux()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`not json`))
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleResolve_ReturnsOriginalURL(t *testing.T) {
	mux := newTestMux()

	w1 := httptest.NewRecorder()
	mux.ServeHTTP(w1, httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"url":"https://example.com"}`)))

	var resp shortenResp
	if err := json.NewDecoder(w1.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/"+resp.ShortCode, nil))

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}

	var resolved resolveResp
	if err := json.NewDecoder(w2.Body).Decode(&resolved); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resolved.OriginalURL != "https://example.com" {
		t.Fatalf("expected original_url %q, got %q", "https://example.com", resolved.OriginalURL)
	}
}

func TestHandleResolve_NotFound(t *testing.T) {
	mux := newTestMux()

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/nonexistent", nil))

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
