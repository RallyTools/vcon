package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func createTestCommand() *cobra.Command {
	cc := NewClientCommand("test", "Tests a connection to vSphere")
	cc.Args = cobra.NoArgs

	cc.RunE = func(_ *cobra.Command, _ []string) error {
		if cc.c.Verbose {
			fmt.Printf("Success\n")
		}

		return nil
	}

	return &cc.Command
}
