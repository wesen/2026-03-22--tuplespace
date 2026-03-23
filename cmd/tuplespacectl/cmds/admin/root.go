package admin

import "github.com/spf13/cobra"

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
	root.AddCommand(healthCmd, spacesCmd, dumpCmd, statsCmd, configCmd, schemaCmd, waitersCmd)
	return root, nil
}
