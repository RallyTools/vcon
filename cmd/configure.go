package cmd

import (
	"encoding/json"
	"errors"

	"github.com/RallyTools/vcon"
	"github.com/spf13/cobra"
)

func createConfigureCommand() *cobra.Command {
	targetIsRef := false

	cc := NewClientCommand("configure TARGET [CONFIGURATION]", "Updates the configuration of a VM")
	cc.Args = cobra.RangeArgs(1, 2)

	cc.RunE = func(_ *cobra.Command, params []string) error {
		target := params[0]

		configuration, err := cc.readString(params[1:])
		if err != nil {
			return err
		}

		vmc := &vcon.VirtualMachineConfiguration{}
		err = json.Unmarshal([]byte(configuration), vmc)
		if err != nil {
			return err
		}

		vm, err := cc.c.FindVM(target, targetIsRef, "network")
		if err != nil {
			return err
		}

		ps, err := cc.c.GetPowerState(vm)
		if err != nil {
			return err
		}

		if ps != vcon.PoweredOff {
			return errors.New("Cannot adjust configuration of a VM that is not powered off")
		}

		err = cc.c.Configure(vm, vmc)
		if err != nil {
			return err
		}

		return nil
	}

	cc.Flags().BoolVar(&targetIsRef, "targetIsRef", targetIsRef, "TARGET parameter is the target VM's uuid")

	return &cc.Command
}
