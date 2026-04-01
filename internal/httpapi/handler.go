package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/bogru/go-url-shortener/internal/shortener"
)

type Handler struct {
	service *shortener.Service
	baseURL string
}

func NewHandler(service *shortener.Service, baseURL string) *Handler {
	return &Handler{
		service: service,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /", h.handleShorten)
	mux.HandleFunc("GET /{shortCode}", h.handleResolve)
}

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	ShortCode     string `json:"short_code"`
	ShortURL      string `json:"short_url"`
	OriginalURL   string `json:"original_url"`
	AlreadyExists bool   `json:"already_exists"`
}

type resolveResponse struct {
	OriginalURL string `json:"original_url"`
}

func (h *Handler) handleShorten(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req shortenRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}

	result, err := h.service.Shorten(req.URL)
	if err != nil {
		if errors.Is(err, shortener.ErrInvalidURL) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to shorten url"})
		return
	}

	statusCode := http.StatusCreated
	if result.AlreadyExists {
		statusCode = http.StatusOK
	}

	writeJSON(w, statusCode, shortenResponse{
		ShortCode:     result.ShortCode,
		ShortURL:      h.baseURL + "/" + result.ShortCode,
		OriginalURL:   result.OriginalURL,
		AlreadyExists: result.AlreadyExists,
	})
}

func (h *Handler) handleResolve(w http.ResponseWriter, r *http.Request) {
	shortCode := r.PathValue("shortCode")
	if shortCode == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "short code is required"})
		return
	}

	originalURL, err := h.service.Resolve(shortCode)
	if err != nil {
		if errors.Is(err, shortener.ErrShortCodeMiss) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve short code"})
		return
	}

	writeJSON(w, http.StatusOK, resolveResponse{OriginalURL: originalURL})
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}
