package admin

import (
	"github.com/spf13/cobra"

	adminTuple "github.com/manuel/wesen/tuplespace/cmd/tuplespacectl/cmds/admin/tuple"
)

func NewCommand() (*cobra.Command, error) {
	root := &cobra.Command{
		Use:   "admin",
		Short: "Administrative tuple space commands",
	}

	healthCmd, err := NewHealthCommand()
	if err != nil {
		return nil, err
	}
	spacesCmd, err := NewSpacesCommand()
	if err != nil {
		return nil, err
	}
	dumpCmd, err := NewDumpCommand()
	if err != nil {
		return nil, err
	}
	peekCmd, err := NewPeekCommand()
	if err != nil {
		return nil, err
	}
	exportCmd, err := NewExportCommand()
	if err != nil {
		return nil, err
	}
	statsCmd, err := NewStatsCommand()
	if err != nil {
		return nil, err
	}
	configCmd, err := NewConfigCommand()
	if err != nil {
		return nil, err
	}
	schemaCmd, err := NewSchemaCommand()
	if err != nil {
		return nil, err
	}
	waitersCmd, err := NewWaitersCommand()
	if err != nil {
		return nil, err
	}
	tupleCmd, err := adminTuple.NewCommand()
	if err != nil {
		return nil, err
	}
	root.AddCommand(healthCmd, spacesCmd, dumpCmd, peekCmd, exportCmd, statsCmd, configCmd, schemaCmd, waitersCmd, tupleCmd)
	return root, nil
}
