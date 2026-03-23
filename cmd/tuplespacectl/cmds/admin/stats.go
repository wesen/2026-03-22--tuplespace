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

type StatsCommand struct {
	*cmds.CommandDescription
}

type StatsSettings struct {
	ServerURL string `glazed:"server-url"`
}

func NewStatsCommand() (*cobra.Command, error) {
	command, err := newStatsCommand()
	if err != nil {
		return nil, err
	}
	return cmdshared.BuildCobraCommand(command)
}

func newStatsCommand() (*StatsCommand, error) {
	desc := cmds.NewCommandDescription(
		"stats",
		cmds.WithShort("Show runtime stats"),
		cmds.WithFlags(
			fields.New("server-url", fields.TypeString, fields.WithDefault("http://127.0.0.1:8080"), fields.WithHelp("TupleSpace server base URL")),
		),
	)
	return &StatsCommand{CommandDescription: desc}, nil
}

func (c *StatsCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := &StatsSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}
	stats, err := client.New(settings.ServerURL).Stats(ctx)
	if err != nil {
		return err
	}
	return gp.AddRow(ctx, glazedtypes.NewRow(
		glazedtypes.MRP("started_at", stats.StartedAt),
		glazedtypes.MRP("uptime_ms", stats.UptimeMS),
		glazedtypes.MRP("space_count", stats.SpaceCount),
		glazedtypes.MRP("tuple_count", stats.TupleCount),
		glazedtypes.MRP("waiter_count", stats.WaiterCount),
		glazedtypes.MRP("notifier_channels", stats.NotifierChannels),
		glazedtypes.MRP("notifier_subscribers", stats.NotifierSubscribers),
		glazedtypes.MRP("notifier_by_channel", stats.NotifierByChannel),
		glazedtypes.MRP("candidate_limit", stats.CandidateLimit),
	))
}
