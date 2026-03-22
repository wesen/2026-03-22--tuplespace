package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/manuel/wesen/tuplespace/internal/api/httpapi"
	"github.com/manuel/wesen/tuplespace/internal/config"
	"github.com/manuel/wesen/tuplespace/internal/migrations"
	"github.com/manuel/wesen/tuplespace/internal/notify"
	"github.com/manuel/wesen/tuplespace/internal/service"
	"github.com/manuel/wesen/tuplespace/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		return err
	}

	db, err := store.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := migrations.ApplyFS(ctx, db, os.DirFS("migrations")); err != nil {
		return err
	}

	notifier, err := notify.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer notifier.Close()

	svc := service.New(db, store.New(), notifier, cfg.CandidateLimit)
	server := &http.Server{
		Addr:    cfg.HTTPListenAddr,
		Handler: httpapi.NewHandler(svc),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownGrace)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
