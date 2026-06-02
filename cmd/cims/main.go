package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cims-go/internal/auth"
	"cims-go/internal/config"
	"cims-go/internal/db"
	web "cims-go/internal/http"
	"cims-go/internal/repositories"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := db.Migrate(cfg.DatabaseURL, "db/migrations"); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.OpenPool(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		return err
	}
	defer pool.Close()

	store := repositories.NewPostgresStore(pool)
	if err := store.EnsureAdmin(ctx, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		return err
	}

	authManager := auth.NewManager(store, cfg.SessionHash, cfg.SessionBlock)
	app, err := web.NewApp(store, authManager)
	if err != nil {
		return err
	}
	app.SetRequestLogging(cfg.RequestLogging)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           app.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("CIMS listening on %s", cfg.Addr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
