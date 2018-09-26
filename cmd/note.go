package cmd

import (
	"github.com/spf13/cobra"
)

const noteLongDescription = `Appends notes to a VM

The "TARGET" argument is a path to the VM.  If the "--targetIsRef" flag is set, the TARGET should be the Mananged Object Reference for the VM.

The "NOTES" argument may be either a path to a file to read in, or a literal string.
If no "NOTES" argument is provided, vcon will read from stdin until is receives an EOF.

Notes will be appended to any existing notes, separated by newlines.  If the "--overwrite" flag is set, all existing notes are replaced.
`

func createNoteCommand() *cobra.Command {
	overwrite := false
	targetIsRef := false

	cc := NewClientCommand("note TARGET [NOTES]", "Appends notes to a VM")
	cc.Args = cobra.RangeArgs(1, 2)
	cc.Long = noteLongDescription

	cc.RunE = func(_ *cobra.Command, params []string) error {
		target := params[0]

		// Get a reader; either Stdin or a specified path
		note, err := cc.readString(params[1:])
		if err != nil {
			return err
		}

		var props []string
		if !overwrite {
			props = []string{"config.annotation"}
		}
		vm, err := cc.c.FindVM(target, targetIsRef, props...)
		if err != nil {
			return err
		}

		err = cc.c.AssignNote(vm, note, overwrite)
		if err != nil {
			return err
		}

		return nil
	}

	cc.Flags().BoolVar(&overwrite, "overwrite", overwrite, "determines whether to replace notes instead of appending")
	cc.Flags().BoolVar(&targetIsRef, "targetIsRef", targetIsRef, "TARGET parameter is the target VM's uuid")

	return &cc.Command
}
