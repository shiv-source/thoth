package cli

import (
	"fmt"
	"os"

	"github.com/shiv-source/thoth/internal/config"
	"github.com/shiv-source/thoth/internal/wiki"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [path]",
		Short: "Scaffold a Thoth wiki directory (default ~/.thoth/wiki)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) == 1 {
				target = args[0]
			} else {
				var err error
				target, err = config.ExpandHome(config.Default().WikiPath)
				if err != nil {
					return err
				}
			}
			expanded, err := config.ExpandHome(target)
			if err != nil {
				return err
			}
			if err := wiki.Scaffold(expanded); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "wiki scaffolded at %s\n", expanded)
			return nil
		},
	}
}
