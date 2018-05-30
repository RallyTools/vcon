package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const destroyLongDescription = `Destroys a VM

The "force" argument will attempt to shut down a VM if it is running.  A running VM cannot be destroyed.
The "TARGET" argument is a path to the VM.  If the "--targetIsRef" flag is set, the TARGET should be the Mananged Object Reference for the VM.`

func createDestroyCommand() *cobra.Command {
	force := false
	targetIsRef := false

	cc := NewClientCommand("destroy TARGET", "Destroys a VM")
	cc.Long = destroyLongDescription
	cc.Args = cobra.ExactArgs(1)

	cc.RunE = func(_ *cobra.Command, params []string) error {
		target := params[0]

		vm, err := cc.c.FindVM(target, targetIsRef)
		if err != nil {
			return err
		}

		if force {
			err = cc.c.EnsureOff(vm)
			if err != nil {
				return err
			}
		}

		err = cc.c.Destroy(vm)
		if err != nil {
			return err
		}

		if cc.c.Verbose {
			fmt.Printf("OK\n")
		}

		return nil
	}

	cc.Flags().BoolVarP(&force, forceKey, "f", force, "will stop a running VM in order to destroy")
	cc.Flags().BoolVar(&targetIsRef, "targetIsRef", targetIsRef, "TARGET parameter is the target VM's uuid")

	return &cc.Command
}
