package cmd

import (
	"fmt"

	"github.com/RallyTools/vcon"
	"github.com/spf13/cobra"
)

func createVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Report the version of the `vcon` application",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Printf("%s\n", vcon.Version)

			return nil
		},
	}

	return cmd
}
