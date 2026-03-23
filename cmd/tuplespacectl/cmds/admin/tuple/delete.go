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

type DeleteCommand struct {
	*cmds.CommandDescription
}

type DeleteSettings struct {
	ServerURL string `glazed:"server-url"`
	TupleID   int    `glazed:"tuple-id"`
}

func NewDeleteCommand() (*cobra.Command, error) {
	command, err := newDeleteCommand()
	if err != nil {
		return nil, err
	}
	return cmdshared.BuildCobraCommand(command)
}

func newDeleteCommand() (*DeleteCommand, error) {
	desc := cmds.NewCommandDescription(
		"delete",
		cmds.WithShort("Delete one tuple by internal tuple id"),
		cmds.WithFlags(
			fields.New("server-url", fields.TypeString, fields.WithDefault("http://127.0.0.1:8080"), fields.WithHelp("TupleSpace server base URL")),
			fields.New("tuple-id", fields.TypeInteger, fields.WithHelp("Tuple id")),
		),
	)
	return &DeleteCommand{CommandDescription: desc}, nil
}

func (c *DeleteCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := &DeleteSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}

	result, err := client.New(settings.ServerURL).DeleteTuple(ctx, int64(settings.TupleID))
	if err != nil {
		return err
	}
	return gp.AddRow(ctx, glazedtypes.NewRow(
		glazedtypes.MRP("tuple_id", result.TupleID),
		glazedtypes.MRP("deleted", result.Deleted),
	))
}
