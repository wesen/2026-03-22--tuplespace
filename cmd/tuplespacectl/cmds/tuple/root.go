package tuple

import (
	"github.com/spf13/cobra"
)

func NewCommand() (*cobra.Command, error) {
	root := &cobra.Command{
		Use:   "tuple",
		Short: "Tuple space tuple operations",
	}

	outCmd, err := NewOutCommand()
	if err != nil {
		return nil, err
	}
	rdCmd, err := NewRdCommand()
	if err != nil {
		return nil, err
	}
	inCmd, err := NewInCommand()
	if err != nil {
		return nil, err
	}

	root.AddCommand(outCmd, rdCmd, inCmd)
	return root, nil
}
