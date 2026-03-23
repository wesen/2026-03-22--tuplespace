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

type PurgeCommand struct {
	*cmds.CommandDescription
}

type PurgeSettings struct {
	ServerURL     string `glazed:"server-url"`
	Space         string `glazed:"space"`
	CreatedBefore string `glazed:"created-before"`
	CreatedAfter  string `glazed:"created-after"`
	Confirm       bool   `glazed:"confirm"`
}

func NewPurgeCommand() (*cobra.Command, error) {
	command, err := newPurgeCommand()
	if err != nil {
		return nil, err
	}
	return cmdshared.BuildCobraCommand(command)
}

func newPurgeCommand() (*PurgeCommand, error) {
	desc := cmds.NewCommandDescription(
		"purge",
		cmds.WithShort("Delete tuples matching admin filters"),
		cmds.WithLong(`Delete tuples matching admin filters.

This command requires --confirm.
`),
		cmds.WithFlags(
			fields.New("server-url", fields.TypeString, fields.WithDefault("http://127.0.0.1:8080"), fields.WithHelp("TupleSpace server base URL")),
			fields.New("space", fields.TypeString, fields.WithHelp("Optional space filter")),
			fields.New("created-before", fields.TypeString, fields.WithHelp("Only delete tuples created before this RFC3339 timestamp")),
			fields.New("created-after", fields.TypeString, fields.WithHelp("Only delete tuples created after this RFC3339 timestamp")),
			fields.New("confirm", fields.TypeBool, fields.WithHelp("Confirm destructive purge execution")),
		),
	)
	return &PurgeCommand{CommandDescription: desc}, nil
}

func (c *PurgeCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := &PurgeSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}

	filter, err := buildTupleFilter(settings.Space, 0, 0, settings.CreatedBefore, settings.CreatedAfter)
	if err != nil {
		return err
	}
	result, err := client.New(settings.ServerURL).Purge(ctx, filter, settings.Confirm)
	if err != nil {
		return err
	}
	return gp.AddRow(ctx, glazedtypes.NewRow(
		glazedtypes.MRP("deleted_count", result.DeletedCount),
	))
}
