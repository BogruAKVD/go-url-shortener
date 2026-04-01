package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/bogru/go-url-shortener/config"
	"github.com/bogru/go-url-shortener/internal/shortener"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func New(ctx context.Context, cfg config.PostgresConfig) (*Store, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.DBName,
	)

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, err
	}

	store := &Store{db: db}
	if err := store.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	s.db.Close()
	return nil
}

func (s *Store) Get(shortCode string) (string, error) {
	const query = `
		SELECT original_url
		FROM short_links
		WHERE short_code = $1
	`

	var originalURL string
	err := s.db.QueryRow(context.Background(), query, shortCode).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", shortener.ErrShortCodeMiss
		}
		return "", err
	}

	return originalURL, nil
}

func (s *Store) Save(shortCode, originalURL string) error {
	const query = `
		INSERT INTO short_links (short_code, original_url)
		VALUES ($1, $2)
	`

	_, err := s.db.Exec(context.Background(), query, shortCode, originalURL)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return shortener.ErrShortCodeBusy
	}

	return err
}

func (s *Store) migrate(ctx context.Context) error {
	const query = `
		CREATE TABLE IF NOT EXISTS short_links (
			short_code VARCHAR(10) PRIMARY KEY,
			original_url TEXT NOT NULL
		)
	`

	_, err := s.db.Exec(ctx, query)
	return err
}
