package main

import (
	"os"

	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	helpcmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/spf13/cobra"

	"github.com/manuel/wesen/tuplespace/cmd/tuplespacectl/cmds/admin"
	"github.com/manuel/wesen/tuplespace/cmd/tuplespacectl/cmds/tuple"
	"github.com/manuel/wesen/tuplespace/cmd/tuplespacectl/doc"
)

func main() {
	root := &cobra.Command{
		Use:   "tuplespacectl",
		Short: "CLI for the TupleSpace service",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.InitLoggerFromCobra(cmd)
		},
	}

	if err := logging.AddLoggingSectionToRootCommand(root, "tuplespacectl"); err != nil {
		cobra.CheckErr(err)
	}

	helpSystem := help.NewHelpSystem()
	if err := doc.AddDocToHelpSystem(helpSystem); err != nil {
		cobra.CheckErr(err)
	}
	helpcmd.SetupCobraRootCommand(helpSystem, root)

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
