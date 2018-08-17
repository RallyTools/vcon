package cmd

import "github.com/spf13/cobra"

func createRelocateCommand() *cobra.Command {
	destination := ""
	name := ""
	targetIsRef := false

	cc := NewClientCommand("relocate TARGET", "Moves and/or renames the TARGET vm")
	cc.Args = cobra.ExactArgs(1)

	cc.RunE = func(_ *cobra.Command, params []string) error {
		target := params[0]

		if name == "" && destination == "" {
			// There is nothing to do here.
			return nil
		}

		vm, err := cc.c.FindVM(target, targetIsRef)
		if err != nil {
			return err
		}

		if name != "" {
			name = cc.generateVMName(name)
		}

		err = cc.c.Relocate(vm, name, destination)
		if err != nil {
			return err
		}

		return nil
	}

	cc.Flags().StringVarP(&destination, destinationKey, "d", destination, "destination folder for VM; if no destination is specified, the VM will not move")

	cc.Flags().StringVarP(&name, nameKey, "n", name, "name of VM; if no name is specified, the name will not change")

	cc.Flags().BoolVar(&targetIsRef, "targetIsRef", targetIsRef, "TARGET parameter is the target VM's uuid")

	return &cc.Command
}
