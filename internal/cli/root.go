package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "dev"

// Execute runs the thoth CLI and returns its error. main only prints and
// exits; all cobra wiring lives here.
func Execute() error {
	return newRootCmd().Execute()
}

// newRootCmd wires the thoth CLI: the serve, init and doctor subcommands plus
// the version command.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "thoth",
		Short:         "Your personal knowledge base, powered by Claude",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newServeCmd(), newInitCmd(), newDoctorCmd())
	root.AddCommand(newVersionCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("thoth", version)
		},
	}
}
