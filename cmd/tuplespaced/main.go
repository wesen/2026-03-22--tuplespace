package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	glazedlogging "github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/help"
	helpcmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	serverdoc "github.com/manuel/wesen/tuplespace/cmd/tuplespaced/doc"
	"github.com/manuel/wesen/tuplespace/internal/api/httpapi"
	"github.com/manuel/wesen/tuplespace/internal/config"
	"github.com/manuel/wesen/tuplespace/internal/migrations"
	"github.com/manuel/wesen/tuplespace/internal/notify"
	"github.com/manuel/wesen/tuplespace/internal/service"
	"github.com/manuel/wesen/tuplespace/internal/store"
)

type ServerCommand struct {
	*cmds.CommandDescription
}

type ServerSettings struct {
	HTTPListenAddr string `glazed:"http-listen-addr"`
	DatabaseURL    string `glazed:"database-url"`
	CandidateLimit int    `glazed:"candidate-limit"`
	ShutdownGrace  string `glazed:"shutdown-grace"`
}

func main() {
	rootCmd, err := newRootCommand()
	cobra.CheckErr(err)
	_ = rootCmd.Execute()
}

func newRootCommand() (*cobra.Command, error) {
	serverCmd, err := newServerCommand()
	if err != nil {
		return nil, err
	}

	rootCmd, err := cli.BuildCobraCommand(serverCmd)
	if err != nil {
		return nil, err
	}
	rootCmd.Use = "tuplespaced"
	rootCmd.Short = "TupleSpace server"
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return glazedlogging.InitLoggerFromCobra(cmd)
	}

	if err := glazedlogging.AddLoggingSectionToRootCommand(rootCmd, "tuplespaced"); err != nil {
		return nil, err
	}

	helpSystem := help.NewHelpSystem()
	if err := serverdoc.AddDocToHelpSystem(helpSystem); err != nil {
		return nil, err
	}
	helpcmd.SetupCobraRootCommand(helpSystem, rootCmd)
	return rootCmd, nil
}

func newServerCommand() (*ServerCommand, error) {
	defaults := config.DefaultsFromEnv()

	desc := cmds.NewCommandDescription(
		"tuplespaced",
		cmds.WithShort("Run the TupleSpace HTTP server"),
		cmds.WithLong(`Run the TupleSpace HTTP server.

This command starts the HTTP API, applies database migrations, and keeps
listening until it receives SIGINT or SIGTERM.
`),
		cmds.WithFlags(
			fields.New("http-listen-addr", fields.TypeString, fields.WithDefault(defaults.HTTPListenAddr), fields.WithHelp("HTTP listen address")),
			fields.New("database-url", fields.TypeString, fields.WithDefault(defaults.DatabaseURL), fields.WithHelp("Postgres connection URL")),
			fields.New("candidate-limit", fields.TypeInteger, fields.WithDefault(defaults.CandidateLimit), fields.WithHelp("Maximum candidate tuples loaded per scan")),
			fields.New("shutdown-grace", fields.TypeString, fields.WithDefault(defaults.ShutdownGrace.String()), fields.WithHelp("Grace period for HTTP shutdown")),
		),
	)

	return &ServerCommand{CommandDescription: desc}, nil
}

func (c *ServerCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := &ServerSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}

	shutdownGrace, err := time.ParseDuration(settings.ShutdownGrace)
	if err != nil {
		return err
	}

	cfg := config.Config{
		HTTPListenAddr: settings.HTTPListenAddr,
		DatabaseURL:    settings.DatabaseURL,
		CandidateLimit: settings.CandidateLimit,
		ShutdownGrace:  shutdownGrace,
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	return runServer(ctx, cfg)
}

func runServer(ctx context.Context, cfg config.Config) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info().
		Str("listen_addr", cfg.HTTPListenAddr).
		Int("candidate_limit", cfg.CandidateLimit).
		Dur("shutdown_grace", cfg.ShutdownGrace).
		Msg("starting tuplespaced")

	db, err := store.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	log.Info().Msg("connected to postgres")

	if err := migrations.ApplyFS(ctx, db, os.DirFS("migrations")); err != nil {
		return err
	}
	log.Info().Msg("applied database migrations")
	migrationFiles, err := migrations.ListFS(os.DirFS("migrations"))
	if err != nil {
		return err
	}

	notifier, err := notify.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer notifier.Close()
	log.Info().Msg("initialized postgres notifier")

	svc := service.New(db, store.New(), notifier, service.Options{
		CandidateLimit: cfg.CandidateLimit,
		StartedAt:      time.Now().UTC(),
		ConfigSnapshot: service.RedactedConfigSnapshot(cfg.HTTPListenAddr, cfg.DatabaseURL, cfg.CandidateLimit, cfg.ShutdownGrace),
		MigrationFiles: migrationFiles,
	})
	server := &http.Server{
		Addr:    cfg.HTTPListenAddr,
		Handler: httpapi.NewHandler(svc),
	}

	go func() {
		<-ctx.Done()
		log.Info().Msg("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownGrace)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("graceful shutdown failed")
			return
		}
		log.Info().Msg("http server shut down cleanly")
	}()

	log.Info().Msg("http server listening")
	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		log.Info().Msg("server closed")
		return nil
	}
	return err
}
