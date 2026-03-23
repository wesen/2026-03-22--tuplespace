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

type WaitersCommand struct {
	*cmds.CommandDescription
}

type WaitersSettings struct {
	ServerURL string `glazed:"server-url"`
}

func NewWaitersCommand() (*cobra.Command, error) {
	command, err := newWaitersCommand()
	if err != nil {
		return nil, err
	}
	return cmdshared.BuildCobraCommand(command)
}

func newWaitersCommand() (*WaitersCommand, error) {
	desc := cmds.NewCommandDescription(
		"waiters",
		cmds.WithShort("Show blocked read or consume operations"),
		cmds.WithFlags(
			fields.New("server-url", fields.TypeString, fields.WithDefault("http://127.0.0.1:8080"), fields.WithHelp("TupleSpace server base URL")),
		),
	)
	return &WaitersCommand{CommandDescription: desc}, nil
}

func (c *WaitersCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := &WaitersSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}
	waiters, err := client.New(settings.ServerURL).Waiters(ctx)
	if err != nil {
		return err
	}
	for _, waiter := range waiters {
		if err := gp.AddRow(ctx, glazedtypes.NewRow(
			glazedtypes.MRP("id", waiter.ID),
			glazedtypes.MRP("space", waiter.Space),
			glazedtypes.MRP("operation", waiter.Operation),
			glazedtypes.MRP("wait_ms", waiter.WaitMS),
			glazedtypes.MRP("started_at", waiter.StartedAt),
			glazedtypes.MRP("template", waiter.Template),
		)); err != nil {
			return err
		}
	}
	return nil
}
