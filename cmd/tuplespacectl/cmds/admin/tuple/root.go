package tuple

import "github.com/spf13/cobra"

func NewCommand() (*cobra.Command, error) {
	root := &cobra.Command{
		Use:   "tuple",
		Short: "Administrative tuple operations",
	}

	getCmd, err := NewGetCommand()
	if err != nil {
		return nil, err
	}
	deleteCmd, err := NewDeleteCommand()
	if err != nil {
		return nil, err
	}
	root.AddCommand(getCmd, deleteCmd)
	return root, nil
}
