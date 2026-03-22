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

type HealthCommand struct {
	*cmds.CommandDescription
}

type HealthSettings struct {
	ServerURL string `glazed:"server-url"`
}

func NewHealthCommand() (*cobra.Command, error) {
	command, err := newHealthCommand()
	if err != nil {
		return nil, err
	}
	return cmdshared.BuildCobraCommand(command)
}

func newHealthCommand() (*HealthCommand, error) {
	desc := cmds.NewCommandDescription(
		"health",
		cmds.WithShort("Check the service health endpoint"),
		cmds.WithFlags(
			fields.New("server-url", fields.TypeString, fields.WithDefault("http://127.0.0.1:8080"), fields.WithHelp("TupleSpace server base URL")),
		),
	)
	return &HealthCommand{CommandDescription: desc}, nil
}

func (c *HealthCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := &HealthSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}

	response, err := client.New(settings.ServerURL).Health(ctx)
	if err != nil {
		return err
	}

	return gp.AddRow(ctx, glazedtypes.NewRow(
		glazedtypes.MRP("ok", response.OK),
		glazedtypes.MRP("server_url", settings.ServerURL),
	))
}
