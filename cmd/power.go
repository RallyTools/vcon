package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	powerOff = "off"
	powerOn  = "on"
	suspend  = "suspend"
)

func createPowerCommand() *cobra.Command {
	targetIsRef := false

	cc := NewClientCommand("power STATE TARGET", "Sets the power state of a VM")
	cc.Args = cobra.ExactArgs(2)
	cc.ValidArgs = []string{"on", "off", "suspend"}

	cc.RunE = func(_ *cobra.Command, params []string) error {
		state := params[0]
		target := params[1]

		vm, err := cc.c.FindVM(target, targetIsRef)
		if err != nil {
			return err
		}

		switch state {
		case powerOff:
			err = cc.c.EnsureOff(vm)
		case powerOn:
			err = cc.c.EnsureOn(vm)
		case suspend:
			err = cc.c.Suspend(vm)
		default:
			err = fmt.Errorf("state '%s' is invalid; must be \"on\", \"off\", or \"suspend\"", state)
		}
		if err != nil {
			return err
		}

		return nil
	}

	cc.Flags().BoolVar(&targetIsRef, "targetIsRef", targetIsRef, "TARGET parameter is the target VM's uuid")

	return &cc.Command
}
