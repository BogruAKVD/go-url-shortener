package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/bogru/go-url-shortener/config"
	"github.com/bogru/go-url-shortener/internal/httpapi"
	"github.com/bogru/go-url-shortener/internal/shortener"
	"github.com/bogru/go-url-shortener/internal/storage/memory"
	"github.com/bogru/go-url-shortener/internal/storage/postgres"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	service, closer, err := buildService(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if closer != nil {
			if err := closer.Close(); err != nil {
				log.Printf("close storage: %v", err)
			}
		}
	}()

	handler := httpapi.NewHandler(service, cfg.BaseURL)

	mux := http.NewServeMux()
	handler.Register(mux)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		log.Printf("starting server on %s with storage=%s", cfg.HTTPAddr, cfg.Storage)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func buildService(cfg config.Config) (*shortener.Service, io.Closer, error) {
	switch cfg.Storage {
	case config.StorageMemory:
		return shortener.NewService(memory.New()), nil, nil
	case config.StoragePostgres:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		store, err := postgres.New(ctx, cfg.Postgres)
		if err != nil {
			return nil, nil, err
		}
		return shortener.NewService(store), store, nil
	default:
		return nil, nil, config.ErrUnsupportedStorage
	}
}
