package admin

import (
	"context"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	glazedtypes "github.com/go-go-golems/glazed/pkg/types"
	"github.com/spf13/cobra"

	cmdshared "github.com/manuel/wesen/tuplespace/cmd/tuplespacectl/cmds"
	"github.com/manuel/wesen/tuplespace/internal/client"
)

type ConfigCommand struct {
	*cmds.CommandDescription
}

type ConfigSettings struct {
	ServerURL string `glazed:"server-url"`
}

func NewConfigCommand() (*cobra.Command, error) {
	command, err := newConfigCommand()
	if err != nil {
		return nil, err
	}
	return cmdshared.BuildCobraCommand(command)
}

func newConfigCommand() (*ConfigCommand, error) {
	desc := cmds.NewCommandDescription(
		"config",
		cmds.WithShort("Show the effective server config"),
		cmds.WithFlags(
			fields.New("server-url", fields.TypeString, fields.WithDefault("http://127.0.0.1:8080"), fields.WithHelp("TupleSpace server base URL")),
		),
	)
	return &ConfigCommand{CommandDescription: desc}, nil
}

func (c *ConfigCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := &ConfigSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}
	cfg, err := client.New(settings.ServerURL).Config(ctx)
	if err != nil {
		return err
	}
	return gp.AddRow(ctx, glazedtypes.NewRow(
		glazedtypes.MRP("http_listen_addr", cfg.HTTPListenAddr),
		glazedtypes.MRP("database_url", cfg.DatabaseURL),
		glazedtypes.MRP("database_host", cfg.DatabaseHost),
		glazedtypes.MRP("database_name", cfg.DatabaseName),
		glazedtypes.MRP("candidate_limit", cfg.CandidateLimit),
		glazedtypes.MRP("shutdown_grace", cfg.ShutdownGrace),
	))
}
