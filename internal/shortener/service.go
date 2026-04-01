package shortener

import (
	"crypto/sha256"
	"errors"
	"net/url"
	"strconv"
)

const (
	shortCodeLength = 10
	alphabet        = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
)

const maxSaltAttempts = 1_000_000

var (
	ErrInvalidURL    = errors.New("invalid url")
	ErrShortCodeMiss = errors.New("short code not found")
	ErrShortCodeBusy = errors.New("short code is already used")
	ErrSaltExhausted = errors.New("could not build unique short code")
)

type Result struct {
	ShortCode     string
	OriginalURL   string
	AlreadyExists bool
}

type Store interface {
	Get(shortCode string) (string, error)
	Save(shortCode, originalURL string) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Shorten(originalURL string) (Result, error) {
	if !isValidURL(originalURL) {
		return Result{}, ErrInvalidURL
	}

	for salt := 0; salt < maxSaltAttempts; salt++ {
		shortCode := buildShortCode(originalURL, salt)
		existingURL, err := s.store.Get(shortCode)
		if err != nil {
			if !errors.Is(err, ErrShortCodeMiss) {
				return Result{}, err
			}

			if err := s.store.Save(shortCode, originalURL); err != nil {
				if errors.Is(err, ErrShortCodeBusy) {
					if saved, getErr := s.store.Get(shortCode); getErr == nil && saved == originalURL {
						return Result{
							ShortCode:     shortCode,
							OriginalURL:   originalURL,
							AlreadyExists: true,
						}, nil
					}
					continue
				}
				return Result{}, err
			}

			return Result{
				ShortCode:     shortCode,
				OriginalURL:   originalURL,
				AlreadyExists: false,
			}, nil
		}

		if existingURL == originalURL {
			return Result{
				ShortCode:     shortCode,
				OriginalURL:   originalURL,
				AlreadyExists: true,
			}, nil
		}
	}

	return Result{}, ErrSaltExhausted
}

func (s *Service) Resolve(shortCode string) (string, error) {
	return s.store.Get(shortCode)
}

func buildShortCode(originalURL string, salt int) string {
	input := originalURL + ":" + strconv.Itoa(salt)
	sum := sha256.Sum256([]byte(input))

	buf := make([]byte, shortCodeLength)
	for i := 0; i < shortCodeLength; i++ {
		buf[i] = alphabet[int(sum[i])%len(alphabet)]
	}

	return string(buf)
}

func isValidURL(raw string) bool {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}

	return parsed.Scheme != "" && parsed.Host != ""
}
