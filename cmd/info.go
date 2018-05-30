package cmd

import (
	"github.com/spf13/cobra"
)

func createInfoCommand() *cobra.Command {
	targetIsRef := false

	cc := NewClientCommand("info TARGET", "Retrieves information about a VM")
	cc.Args = cobra.ExactArgs(1)

	cc.RunE = func(_ *cobra.Command, params []string) error {
		target := params[0]

		vm, err := cc.c.FindVM(target, targetIsRef, "network", "summary")
		if err != nil {
			return err
		}

		return cc.writeVMInfoToConsole(vm)
	}

	cc.Flags().BoolVar(&targetIsRef, "targetIsRef", targetIsRef, "TARGET parameter is the target VM's uuid")

	return &cc.Command
}
