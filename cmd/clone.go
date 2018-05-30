package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func createCloneCommand() *cobra.Command {
	name := ""
	on := true

	cc := NewClientCommand("clone SOURCE", "Clones a template or VM")
	cc.Args = cobra.ExactArgs(1)

	cc.RunE = func(_ *cobra.Command, params []string) error {
		source := params[0]

		vm, err := cc.c.FindVM(source, false)
		if err != nil {
			return err
		}

		name := cc.generateVMName(name)
		destination := viper.GetString(destinationKey)
		resourcePool := viper.GetString(resourcePoolKey)

		newVM, err := cc.c.Clone(vm, name, destination, resourcePool)
		if err != nil {
			return err
		}

		if on {
			cc.c.EnsureOn(newVM)
			if err != nil {
				return fmt.Errorf("Error requesting power-on new VM: %s", err.Error())
			}
		}

		return cc.writeVMInfoToConsole(newVM)
	}

	cc.Flags().StringP(destinationKey, "d", "", "destination folder for new VM")
	viper.BindPFlag(destinationKey, cc.Flags().Lookup(destinationKey))

	cc.Flags().StringVarP(&name, nameKey, "n", name, "name of new VM; if no name is specified, one will be generated.")

	cc.Flags().BoolVar(&on, "on", true, "determines whether the VM will be started after cloning")

	cc.Flags().String(resourcePoolKey, "", "resource pool name for new VM")
	viper.BindPFlag(resourcePoolKey, cc.Flags().Lookup(resourcePoolKey))

	return &cc.Command
}
