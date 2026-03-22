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
	root.AddCommand(healthCmd)
	return root, nil
}
