package admin

import (
	"context"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	glazedtypes "github.com/go-go-golems/glazed/pkg/types"
	"github.com/spf13/cobra"

	cmdshared "github.com/manuel/wesen/tuplespace/cmd/tuplespacectl/cmds"
	"github.com/manuel/wesen/tuplespace/internal/admin"
	"github.com/manuel/wesen/tuplespace/internal/client"
)

type DumpCommand struct {
	*cmds.CommandDescription
}

type DumpSettings struct {
	ServerURL     string `glazed:"server-url"`
	Space         string `glazed:"space"`
	Limit         int    `glazed:"limit"`
	Offset        int    `glazed:"offset"`
	CreatedBefore string `glazed:"created-before"`
	CreatedAfter  string `glazed:"created-after"`
}

func NewDumpCommand() (*cobra.Command, error) {
	command, err := newDumpCommand()
	if err != nil {
		return nil, err
	}
	return cmdshared.BuildCobraCommand(command)
}

func newDumpCommand() (*DumpCommand, error) {
	desc := cmds.NewCommandDescription(
		"dump",
		cmds.WithShort("Dump tuples from one space or all spaces"),
		cmds.WithFlags(
			fields.New("server-url", fields.TypeString, fields.WithDefault("http://127.0.0.1:8080"), fields.WithHelp("TupleSpace server base URL")),
			fields.New("space", fields.TypeString, fields.WithHelp("Optional space filter")),
			fields.New("limit", fields.TypeInteger, fields.WithHelp("Maximum tuples to return")),
			fields.New("offset", fields.TypeInteger, fields.WithHelp("Tuple offset")),
			fields.New("created-before", fields.TypeString, fields.WithHelp("Only include tuples created before this RFC3339 timestamp")),
			fields.New("created-after", fields.TypeString, fields.WithHelp("Only include tuples created after this RFC3339 timestamp")),
		),
	)
	return &DumpCommand{CommandDescription: desc}, nil
}

func (c *DumpCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := &DumpSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}

	filter, err := dumpFilterFromSettings(settings)
	if err != nil {
		return err
	}

	tuples, err := client.New(settings.ServerURL).Dump(ctx, filter)
	if err != nil {
		return err
	}
	for _, tuple := range tuples {
		if err := gp.AddRow(ctx, glazedtypes.NewRow(
			glazedtypes.MRP("id", tuple.ID),
			glazedtypes.MRP("space", tuple.Space),
			glazedtypes.MRP("arity", tuple.Arity),
			glazedtypes.MRP("created_at", tuple.CreatedAt),
			glazedtypes.MRP("tuple", tuple.Tuple),
		)); err != nil {
			return err
		}
	}
	return nil
}

func dumpFilterFromSettings(settings *DumpSettings) (admin.TupleFilter, error) {
	filter := admin.TupleFilter{
		Space:  settings.Space,
		Limit:  settings.Limit,
		Offset: settings.Offset,
	}
	if settings.CreatedBefore != "" {
		t, err := time.Parse(time.RFC3339, settings.CreatedBefore)
		if err != nil {
			return admin.TupleFilter{}, err
		}
		filter.CreatedBefore = &t
	}
	if settings.CreatedAfter != "" {
		t, err := time.Parse(time.RFC3339, settings.CreatedAfter)
		if err != nil {
			return admin.TupleFilter{}, err
		}
		filter.CreatedAfter = &t
	}
	return filter, nil
}
