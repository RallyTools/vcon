package cmd

import (
	"errors"

	"github.com/RallyTools/vcon"
	"github.com/spf13/cobra"
)

func createSnapshotCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot [create|remove|revert]",
		Short: "EXPERIMENTAL: Manipulates snapshots for a VM",
	}

	cmd.AddCommand(
		createSnapshotCreateCommand(),
		createSnapshotListCommand(),
		createSnapshotRemoveCommand(),
		createSnapshotRevertCommand(),
	)

	return cmd
}

func createSnapshotCreateCommand() *cobra.Command {
	name := ""
	targetIsRef := false

	cc := NewClientCommand("create TARGET", "Creates a snapshot of a VM")
	cc.Args = cobra.ExactArgs(1)

	cc.RunE = func(_ *cobra.Command, params []string) error {
		target := params[0]

		vm, err := cc.c.FindVM(target, targetIsRef)
		if err != nil {
			return err
		}

		// Should check that we are powered off.
		ps, err := cc.c.GetPowerState(vm)
		if err != nil {
			return err
		}
		if ps != vcon.PoweredOff {
			return errors.New("Cannot get a snapshot of a running machine")
		}

		name := cc.generateSnapshotName(name)
		err = cc.c.SnapshotCreate(vm, name)
		if err != nil {
			return err
		}

		return nil
	}

	cc.Flags().StringVarP(&name, nameKey, "n", name, "name of new snapshot; if no name is specified, one will be generated.")
	cc.Flags().BoolVar(&targetIsRef, "targetIsRef", targetIsRef, "TARGET parameter is the target VM's uuid")

	return &cc.Command
}

func createSnapshotListCommand() *cobra.Command {
	targetIsRef := false

	cc := NewClientCommand("list TARGET", "lists all snapshots of a VM")
	cc.Args = cobra.ExactArgs(1)

	cc.RunE = func(_ *cobra.Command, params []string) error {
		target := params[0]

		vm, err := cc.c.FindVM(target, targetIsRef, "snapshot")
		if err != nil {
			return err
		}

		cc.c.SnapshotList(vm)

		return nil
	}

	cc.Flags().BoolVar(&targetIsRef, "targetIsRef", targetIsRef, "TARGET parameter is the target VM's uuid")

	return &cc.Command
}

const longSnapshotRemoveDescription = `Removes one or all of the snapshots on a Virual Machine

The "TARGET" argument is a path to the VM.  If the "--targetIsRef" flag is set, the TARGET should be the Mananged Object Reference for the VM.

The "SNAPSHOT" argument is optional.  If not provided, all of the VM's snapshots will be removed.  If the argument is present, it is the name of a snapshot, unless the "--snapshotIsRef" flag is set, in which case the argument is the snapshot's unique ID.  If multiple snapshots have the specified name, the request will fail.`

func createSnapshotRemoveCommand() *cobra.Command {
	snapshotIsRef := false
	targetIsRef := false

	cc := NewClientCommand("remove TARGET [SNAPSHOT]", "Removes one or all of the snapshots on a Virual Machine")
	cc.Args = cobra.RangeArgs(2, 3)
	cc.Long = longSnapshotRemoveDescription

	cc.RunE = func(_ *cobra.Command, params []string) error {
		target := params[0]
		snapshot := ""
		if len(params) == 2 {
			snapshot = params[1]
		}

		vm, err := cc.c.FindVM(target, targetIsRef)
		if err != nil {
			return err
		}

		if snapshot != "" {
			moRef, err := cc.c.FindSnapshot(vm, snapshot, snapshotIsRef)
			if err != nil {
				return err
			}

			err = cc.c.SnapshotRemove(vm, moRef)
			if err != nil {
				return err
			}
		} else {
			err = cc.c.SnapshotRemoveAll(vm)
			if err != nil {
				return err
			}
		}

		return nil
	}

	cc.Flags().BoolVar(&snapshotIsRef, "snapshotIsRef", snapshotIsRef, "SNAPSHOT parameter is the snapshot's uuid")
	cc.Flags().BoolVar(&targetIsRef, "targetIsRef", targetIsRef, "TARGET parameter is the target VM's uuid")

	return &cc.Command
}

const longSnapshotRevertDescription = `Reverts the VM to a snapshot state

The "TARGET" argument is a path to the VM.  If the "--targetIsRef" flag is set, the TARGET should be the Mananged Object Reference for the VM.

The "SNAPSHOT" argument is optional.  If not provided, this will revert the VM to it's current snapshot state.  If the argument is present, it is the name of a snapshot, unless the "--snapshotIsRef" flag is set, in which case the argument is the snapshot's unique ID.  If multiple snapshots have the specified name, the request will fail.`

func createSnapshotRevertCommand() *cobra.Command {
	snapshotIsRef := false
	targetIsRef := false

	cc := NewClientCommand("revert TARGET [SNAPSHOT]", "reverts a VM to a snapshot")
	cc.Args = cobra.RangeArgs(1, 2)

	cc.RunE = func(_ *cobra.Command, params []string) error {
		target := params[0]

		vm, err := cc.c.FindVM(target, targetIsRef)
		if err != nil {
			return err
		}

		if len(params) == 1 {
			err = cc.c.SnapshotRevert(vm)
		} else {
			name := params[1]
			snapshot, err := cc.c.FindSnapshot(vm, name, snapshotIsRef)
			if err != nil {
				return err
			}

			err = cc.c.SnapshotRevertTo(vm, snapshot)
		}
		if err != nil {
			return err
		}

		return nil
	}

	cc.Flags().BoolVar(&snapshotIsRef, "snapshotIsRef", snapshotIsRef, "SNAPSHOT parameter is the snapshot's uuid")
	cc.Flags().BoolVar(&targetIsRef, "targetIsRef", targetIsRef, "TARGET parameter is the target VM's uuid")

	return &cc.Command
}
