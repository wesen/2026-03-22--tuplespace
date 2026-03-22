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

type InCommand struct {
	*cmds.CommandDescription
}

type InSettings struct {
	ServerURL        string `glazed:"server-url"`
	Space            string `glazed:"space"`
	TemplateJSONFile string `glazed:"template-json-file"`
	TemplateSpec     string `glazed:"template-spec"`
	WaitMS           int    `glazed:"wait-ms"`
}

func NewInCommand() (*cobra.Command, error) {
	command, err := newInCommand()
	if err != nil {
		return nil, err
	}
	return cli.BuildCobraCommandFromCommand(command)
}

func newInCommand() (*InCommand, error) {
	desc := cmds.NewCommandDescription(
		"in",
		cmds.WithShort("Consume a matching tuple"),
		cmds.WithLong(`Consume a matching tuple.

Provide either --template-json-file with JSON or --template-spec with the compact DSL.

Examples:
  tuplespacectl tuple in --space jobs --template-spec 'job,?id:int'
  tuplespacectl tuple in --space jobs --template-spec '("job with spaces",?id:int,false)'
`),
		cmds.WithFlags(
			fields.New("server-url", fields.TypeString, fields.WithDefault("http://127.0.0.1:8080"), fields.WithHelp("TupleSpace server base URL")),
			fields.New("space", fields.TypeString, fields.WithHelp("Tuple space name")),
			fields.New("template-json-file", fields.TypeString, fields.WithHelp("Path to a template JSON file")),
			fields.New("template-spec", fields.TypeString, fields.WithHelp("Compact template DSL, for example: job,?id:int")),
			fields.New("wait-ms", fields.TypeInteger, fields.WithDefault(0), fields.WithHelp("How long to wait for a matching tuple")),
		),
	)
	return &InCommand{CommandDescription: desc}, nil
}

func (c *InCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := &InSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}

	template, err := sharedcmds.LoadTemplateInput(settings.TemplateJSONFile, settings.TemplateSpec)
	if err != nil {
		return err
	}

	response, err := client.New(settings.ServerURL).In(ctx, settings.Space, template, int64(settings.WaitMS))
	if err != nil {
		return err
	}

	return gp.AddRow(ctx, glazedtypes.NewRow(
		glazedtypes.MRP("ok", response.OK),
		glazedtypes.MRP("space", settings.Space),
		glazedtypes.MRP("tuple", response.Tuple),
		glazedtypes.MRP("bindings", response.Bindings),
	))
}
