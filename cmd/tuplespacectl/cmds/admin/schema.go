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

type SchemaCommand struct {
	*cmds.CommandDescription
}

type SchemaSettings struct {
	ServerURL string `glazed:"server-url"`
}

func NewSchemaCommand() (*cobra.Command, error) {
	command, err := newSchemaCommand()
	if err != nil {
		return nil, err
	}
	return cmdshared.BuildCobraCommand(command)
}

func newSchemaCommand() (*SchemaCommand, error) {
	desc := cmds.NewCommandDescription(
		"schema",
		cmds.WithShort("Show migration files and detected schema objects"),
		cmds.WithFlags(
			fields.New("server-url", fields.TypeString, fields.WithDefault("http://127.0.0.1:8080"), fields.WithHelp("TupleSpace server base URL")),
		),
	)
	return &SchemaCommand{CommandDescription: desc}, nil
}

func (c *SchemaCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := &SchemaSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}
	schemaInfo, err := client.New(settings.ServerURL).Schema(ctx)
	if err != nil {
		return err
	}
	return gp.AddRow(ctx, glazedtypes.NewRow(
		glazedtypes.MRP("migration_files", schemaInfo.MigrationFiles),
		glazedtypes.MRP("tables", schemaInfo.Tables),
		glazedtypes.MRP("indexes", schemaInfo.Indexes),
		glazedtypes.MRP("missing_tables", schemaInfo.MissingTables),
		glazedtypes.MRP("missing_indexes", schemaInfo.MissingIndexes),
	))
}
