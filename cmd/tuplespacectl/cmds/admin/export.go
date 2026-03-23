package admin

import (
	"context"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/spf13/cobra"

	cmdshared "github.com/manuel/wesen/tuplespace/cmd/tuplespacectl/cmds"
	"github.com/manuel/wesen/tuplespace/internal/client"
)

type ExportCommand struct {
	*cmds.CommandDescription
}

type ExportSettings struct {
	ServerURL     string `glazed:"server-url"`
	Space         string `glazed:"space"`
	Limit         int    `glazed:"limit"`
	Offset        int    `glazed:"offset"`
	CreatedBefore string `glazed:"created-before"`
	CreatedAfter  string `glazed:"created-after"`
}

func NewExportCommand() (*cobra.Command, error) {
	command, err := newExportCommand()
	if err != nil {
		return nil, err
	}
	return cmdshared.BuildCobraCommand(command)
}

func newExportCommand() (*ExportCommand, error) {
	desc := cmds.NewCommandDescription(
		"export",
		cmds.WithShort("Export tuples with admin filters"),
		cmds.WithFlags(
			fields.New("server-url", fields.TypeString, fields.WithDefault("http://127.0.0.1:8080"), fields.WithHelp("TupleSpace server base URL")),
			fields.New("space", fields.TypeString, fields.WithHelp("Optional space filter")),
			fields.New("limit", fields.TypeInteger, fields.WithHelp("Maximum tuples to return")),
			fields.New("offset", fields.TypeInteger, fields.WithHelp("Tuple offset")),
			fields.New("created-before", fields.TypeString, fields.WithHelp("Only include tuples created before this RFC3339 timestamp")),
			fields.New("created-after", fields.TypeString, fields.WithHelp("Only include tuples created after this RFC3339 timestamp")),
		),
	)
	return &ExportCommand{CommandDescription: desc}, nil
}

func (c *ExportCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := &ExportSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}

	filter, err := buildTupleFilter(settings.Space, settings.Limit, settings.Offset, settings.CreatedBefore, settings.CreatedAfter)
	if err != nil {
		return err
	}
	tuples, err := client.New(settings.ServerURL).Export(ctx, filter)
	if err != nil {
		return err
	}
	return addTupleRows(ctx, gp, tuples)
}
