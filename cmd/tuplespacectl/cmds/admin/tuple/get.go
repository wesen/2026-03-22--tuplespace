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

	cmdshared "github.com/manuel/wesen/tuplespace/cmd/tuplespacectl/cmds"
	"github.com/manuel/wesen/tuplespace/internal/client"
)

type GetCommand struct {
	*cmds.CommandDescription
}

type GetSettings struct {
	ServerURL string `glazed:"server-url"`
	TupleID   int    `glazed:"tuple-id"`
}

func NewGetCommand() (*cobra.Command, error) {
	command, err := newGetCommand()
	if err != nil {
		return nil, err
	}
	return cmdshared.BuildCobraCommand(command)
}

func newGetCommand() (*GetCommand, error) {
	desc := cmds.NewCommandDescription(
		"get",
		cmds.WithShort("Fetch one tuple by internal tuple id"),
		cmds.WithFlags(
			fields.New("server-url", fields.TypeString, fields.WithDefault("http://127.0.0.1:8080"), fields.WithHelp("TupleSpace server base URL")),
			fields.New("tuple-id", fields.TypeInteger, fields.WithHelp("Tuple id")),
		),
	)
	return &GetCommand{CommandDescription: desc}, nil
}

func (c *GetCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := &GetSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}

	record, err := client.New(settings.ServerURL).GetTuple(ctx, int64(settings.TupleID))
	if err != nil {
		return err
	}
	return gp.AddRow(ctx, glazedtypes.NewRow(
		glazedtypes.MRP("id", record.ID),
		glazedtypes.MRP("space", record.Space),
		glazedtypes.MRP("arity", record.Arity),
		glazedtypes.MRP("created_at", record.CreatedAt),
		glazedtypes.MRP("tuple", record.Tuple),
	))
}
