package tuple

import (
	"context"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	glazedtypes "github.com/go-go-golems/glazed/pkg/types"
	"github.com/spf13/cobra"

	sharedcmds "github.com/manuel/wesen/tuplespace/cmd/tuplespacectl/cmds"
	"github.com/manuel/wesen/tuplespace/internal/client"
)

type OutCommand struct {
	*cmds.CommandDescription
}

type OutSettings struct {
	ServerURL string `glazed:"server-url"`
	Space     string `glazed:"space"`
	TupleFile string `glazed:"tuple-file"`
	TupleSpec string `glazed:"tuple-spec"`
}

func NewOutCommand() (*cobra.Command, error) {
	command, err := newOutCommand()
	if err != nil {
		return nil, err
	}
	return cli.BuildCobraCommandFromCommand(command)
}

func newOutCommand() (*OutCommand, error) {
	desc := cmds.NewCommandDescription(
		"out",
		cmds.WithShort("Write a tuple to the tuple space"),
		cmds.WithLong(`Write a tuple to the tuple space.

Provide either --tuple-file with JSON or --tuple-spec with the compact DSL.

Examples:
  tuplespacectl tuple out --space jobs --tuple-spec 'job,42,true'
  tuplespacectl tuple out --space jobs --tuple-spec '("job with spaces",42,false)'
`),
		cmds.WithFlags(
			fields.New("server-url", fields.TypeString, fields.WithDefault("http://127.0.0.1:8080"), fields.WithHelp("TupleSpace server base URL")),
			fields.New("space", fields.TypeString, fields.WithHelp("Tuple space name")),
			fields.New("tuple-file", fields.TypeString, fields.WithHelp("Path to a tuple JSON file")),
			fields.New("tuple-spec", fields.TypeString, fields.WithHelp("Compact tuple DSL, for example: job,42,true")),
		),
	)
	return &OutCommand{CommandDescription: desc}, nil
}

func (c *OutCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := &OutSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}

	tuple, err := sharedcmds.LoadTupleInput(settings.TupleFile, settings.TupleSpec)
	if err != nil {
		return err
	}

	response, err := client.New(settings.ServerURL).Out(ctx, settings.Space, tuple)
	if err != nil {
		return err
	}

	return gp.AddRow(ctx, glazedtypes.NewRow(
		glazedtypes.MRP("ok", response.OK),
		glazedtypes.MRP("space", response.Space),
		glazedtypes.MRP("arity", response.Arity),
	))
}
