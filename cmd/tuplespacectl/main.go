package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/manuel/wesen/tuplespace/cmd/tuplespacectl/cmds/admin"
	"github.com/manuel/wesen/tuplespace/cmd/tuplespacectl/cmds/tuple"
)

func main() {
	root := &cobra.Command{
		Use:   "tuplespacectl",
		Short: "CLI for the TupleSpace service",
	}

	tupleCmd, err := tuple.NewCommand()
	if err != nil {
		panic(err)
	}
	adminCmd, err := admin.NewCommand()
	if err != nil {
		panic(err)
	}
	root.AddCommand(tupleCmd, adminCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
