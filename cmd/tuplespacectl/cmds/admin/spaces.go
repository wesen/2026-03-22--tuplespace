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

type SpacesCommand struct {
	*cmds.CommandDescription
}

type SpacesSettings struct {
	ServerURL string `glazed:"server-url"`
}

func NewSpacesCommand() (*cobra.Command, error) {
	command, err := newSpacesCommand()
	if err != nil {
		return nil, err
	}
	return cmdshared.BuildCobraCommand(command)
}

func newSpacesCommand() (*SpacesCommand, error) {
	desc := cmds.NewCommandDescription(
		"spaces",
		cmds.WithShort("List spaces with tuple counts"),
		cmds.WithFlags(
			fields.New("server-url", fields.TypeString, fields.WithDefault("http://127.0.0.1:8080"), fields.WithHelp("TupleSpace server base URL")),
		),
	)
	return &SpacesCommand{CommandDescription: desc}, nil
}

func (c *SpacesCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := &SpacesSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}

	spaces, err := client.New(settings.ServerURL).Spaces(ctx)
	if err != nil {
		return err
	}
	for _, summary := range spaces {
		if err := gp.AddRow(ctx, glazedtypes.NewRow(
			glazedtypes.MRP("space", summary.Space),
			glazedtypes.MRP("tuple_count", summary.TupleCount),
			glazedtypes.MRP("oldest_tuple_at", summary.OldestTupleAt),
			glazedtypes.MRP("newest_tuple_at", summary.NewestTupleAt),
		)); err != nil {
			return err
		}
	}
	return nil
}
