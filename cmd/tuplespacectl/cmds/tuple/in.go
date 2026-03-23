package tuple

import (
	"context"

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
	ServerURL        string   `glazed:"server-url"`
	Space            string   `glazed:"space"`
	TemplateJSONFile string   `glazed:"template-json-file"`
	TemplateSpec     string   `glazed:"template-spec"`
	TemplateSpecs    []string `glazed:"template-specs"`
	WaitMS           int      `glazed:"wait-ms"`
}

func NewInCommand() (*cobra.Command, error) {
	command, err := newInCommand()
	if err != nil {
		return nil, err
	}
	return sharedcmds.BuildCobraCommand(command)
}

func newInCommand() (*InCommand, error) {
	desc := cmds.NewCommandDescription(
		"in",
		cmds.WithShort("Consume a matching tuple"),
		cmds.WithLong(`Consume a matching tuple.

Provide either --template-json-file with JSON or --template-spec with the compact DSL.
You can also pass multiple template specs as positional arguments.

Examples:
  tuplespacectl tuple in --space jobs --template-spec 'job,?id:int'
  tuplespacectl tuple in --space jobs --template-spec '("job with spaces",?id:int,false)'
  tuplespacectl tuple in --space jobs 'job,?id:int' 'worker,?id:int'
`),
		cmds.WithFlags(
			fields.New("server-url", fields.TypeString, fields.WithDefault("http://127.0.0.1:8080"), fields.WithHelp("TupleSpace server base URL")),
			fields.New("space", fields.TypeString, fields.WithHelp("Tuple space name")),
			fields.New("template-json-file", fields.TypeString, fields.WithHelp("Path to a template JSON file")),
			fields.New("template-spec", fields.TypeString, fields.WithHelp("Compact template DSL, for example: job,?id:int")),
			fields.New("wait-ms", fields.TypeInteger, fields.WithDefault(0), fields.WithHelp("How long to wait for a matching tuple")),
		),
		cmds.WithArguments(
			fields.New("template-specs", fields.TypeStringList, fields.WithHelp("One or more template specs as positional arguments")),
		),
	)
	return &InCommand{CommandDescription: desc}, nil
}

func (c *InCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := &InSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}

	templates, err := sharedcmds.LoadTemplateInputs(settings.TemplateJSONFile, settings.TemplateSpec, settings.TemplateSpecs)
	if err != nil {
		return err
	}

	cliClient := client.NewWithTimeout(settings.ServerURL, client.TimeoutForWaitMS(int64(settings.WaitMS)))
	for i, template := range templates {
		response, err := cliClient.In(ctx, settings.Space, template, int64(settings.WaitMS))
		if err != nil {
			return err
		}

		if err := gp.AddRow(ctx, glazedtypes.NewRow(
			glazedtypes.MRP("index", i),
			glazedtypes.MRP("ok", response.OK),
			glazedtypes.MRP("space", settings.Space),
			glazedtypes.MRP("tuple", response.Tuple),
			glazedtypes.MRP("bindings", response.Bindings),
		)); err != nil {
			return err
		}
	}
	return nil
}
