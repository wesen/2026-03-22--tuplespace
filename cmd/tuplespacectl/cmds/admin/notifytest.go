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

type NotifyTestCommand struct {
	*cmds.CommandDescription
}

type NotifyTestSettings struct {
	ServerURL string `glazed:"server-url"`
	Space     string `glazed:"space"`
}

func NewNotifyTestCommand() (*cobra.Command, error) {
	command, err := newNotifyTestCommand()
	if err != nil {
		return nil, err
	}
	return cmdshared.BuildCobraCommand(command)
}

func newNotifyTestCommand() (*NotifyTestCommand, error) {
	desc := cmds.NewCommandDescription(
		"notify-test",
		cmds.WithShort("Send a test notifier wakeup for one space"),
		cmds.WithFlags(
			fields.New("server-url", fields.TypeString, fields.WithDefault("http://127.0.0.1:8080"), fields.WithHelp("TupleSpace server base URL")),
			fields.New("space", fields.TypeString, fields.WithHelp("Target space")),
		),
	)
	return &NotifyTestCommand{CommandDescription: desc}, nil
}

func (c *NotifyTestCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := &NotifyTestSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}

	result, err := client.New(settings.ServerURL).NotifyTest(ctx, settings.Space)
	if err != nil {
		return err
	}
	return gp.AddRow(ctx, glazedtypes.NewRow(
		glazedtypes.MRP("space", result.Space),
		glazedtypes.MRP("channel", result.Channel),
		glazedtypes.MRP("subscriber_count", result.SubscriberCount),
		glazedtypes.MRP("channel_subscriber_count", result.ChannelSubscriberCount),
		glazedtypes.MRP("notifier_channels", result.NotifierChannels),
		glazedtypes.MRP("notifier_by_channel", result.NotifierByChannel),
	))
}
