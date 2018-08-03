package vcon

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/types"
)

// Snapshot represents a node in a hierarchical list of VM snapshots
type Snapshot struct {
	Name     string     `json:"name"`
	Ref      string     `json:"ref"`
	Children []Snapshot `json:"children,omitempty"`
}

// FindSnapshot will locate the Managed Object Reference for a snapshot, either
// by looking it up by name, or converting the provided ref into a MORef.
func (c *Client) FindSnapshot(vm *VirtualMachine, name string, byRef bool) (*types.ManagedObjectReference, error) {
	var moRef *types.ManagedObjectReference

	if byRef {
		moRef = &types.ManagedObjectReference{
			Type:  "VirtualMachineSnapshot",
			Value: name,
		}
	} else {
		err := func() error {
			ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
			defer cancelFn()

			var err error
			moRef, err = vm.VM.FindSnapshot(ctx, name)
			if err := c.checkErr(ctx, err); err != nil {
				return errors.Wrapf(err, "While finding snapshot")
			}

			return nil
		}()

		if err != nil {
			switch err := errors.Cause(err).(type) {
			case *TimeoutExceededError:
				// handle specifically
				return nil, fmt.Errorf("Timeout while attempting to find snapshot for a VM")
			default:
				// unknown error
				return nil, errors.Wrap(err, "Got error while finding snapshot for a VM")
			}
		}
	}

	return moRef, nil
}

// SnapshotCreate will create a snapshot of the current VM.  It is assumed that the
// VM is already powered off.  Taking a snapshot of a powered-on or suspended VM
// _may_ be successful, but certain configurations will cause problems, and we are
// not snapshotting the current memory state or quiescing the file system.
// See https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.vm_admin.doc/GUID-53F65726-A23B-4CF0-A7D5-48E584B88613.html
func (c *Client) SnapshotCreate(vm *VirtualMachine, name string) (*types.ManagedObjectReference, error) {
	if c.Verbose {
		fmt.Printf("Creating a VM snapshot...\n")
	}

	var res types.ManagedObjectReference

	err := func() error {
		ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
		defer cancelFn()

		task, err := vm.VM.CreateSnapshot(ctx, name, "", false, false)
		any, err := c.finishTask(ctx, task, err)
		if err != nil {
			return errors.Wrapf(err, "While snapshotting VM")
		}

		res = any.(types.ManagedObjectReference)

		return nil
	}()

	if err != nil {
		switch err := errors.Cause(err).(type) {
		case *TimeoutExceededError:
			// handle specifically
			return nil, fmt.Errorf("Timeout while attempting to snapshot VM")
		default:
			// unknown error
			return nil, errors.Wrap(err, "Got error while snapshotting a VM")
		}
	}

	return &res, nil
}

// SnapshotList retrieves all the snapshots for the provided VM, hierarchically
func (c *Client) SnapshotList(vm *VirtualMachine) (*Snapshot, error) {
	if c.Verbose {
		fmt.Printf("Getting a list of snapshots for VM...\n")
	}

	var sn *Snapshot

	err := func() error {
		_, cancelFn := context.WithTimeout(context.Background(), c.timeout)
		defer cancelFn()

		if vm.MO.Snapshot == nil {
			return nil
		}

		// TODO: unclear how to proceed here.

		return nil
	}()

	if err != nil {
		switch err := errors.Cause(err).(type) {
		case *TimeoutExceededError:
			// handle specifically
			return nil, fmt.Errorf("Timeout while attempting to list snapshots for a VM")
		default:
			// unknown error
			return nil, errors.Wrap(err, "Got error while listing snapshots for a VM")
		}
	}

	return sn, nil
}

// SnapshotRemove removes a single snapshot from the provided VM
func (c *Client) SnapshotRemove(vm *VirtualMachine, moRef *types.ManagedObjectReference) error {
	err := func() error {
		ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
		defer cancelFn()

		consolidate := true
		req := types.RemoveSnapshot_Task{
			This:           moRef.Reference(),
			RemoveChildren: false,
			Consolidate:    &consolidate,
		}

		res, err := methods.RemoveSnapshot_Task(ctx, vm.VM.Client(), &req)
		if err := c.checkErr(ctx, err); err != nil {
			return errors.Wrapf(err, "While removing snapshot")
		}

		task := object.NewTask(vm.VM.Client(), res.Returnval)
		_, err = c.finishTask(ctx, task, nil)
		if err != nil {
			return errors.Wrapf(err, "While waiting for snapshot removal")
		}

		return nil
	}()

	if err != nil {
		switch err := errors.Cause(err).(type) {
		case *TimeoutExceededError:
			// handle specifically
			return fmt.Errorf("Timeout while attempting to list snapshots for a VM")
		default:
			// unknown error
			return errors.Wrap(err, "Got error while listing snapshots for a VM")
		}
	}

	return nil
}

// SnapshotRemoveAll removes snapshots from the provided VM
func (c *Client) SnapshotRemoveAll(vm *VirtualMachine) error {
	err := func() error {
		ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
		defer cancelFn()

		consolidate := true
		task, err := vm.VM.RemoveAllSnapshot(ctx, &consolidate)
		_, err = c.finishTask(ctx, task, err)
		if err != nil {
			return errors.Wrapf(err, "While removing all snapshots")
		}

		return nil
	}()

	if err != nil {
		switch err := errors.Cause(err).(type) {
		case *TimeoutExceededError:
			// handle specifically
			return fmt.Errorf("Timeout while attempting to remove all snapshots for a VM")
		default:
			// unknown error
			return errors.Wrap(err, "Got error while removing all snapshots for a VM")
		}
	}

	return nil
}

// SnapshotRevert will revert the provided VM back the previous snapshot
func (c *Client) SnapshotRevert(vm *VirtualMachine) error {
	err := func() error {
		ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
		defer cancelFn()

		task, err := vm.VM.RevertToCurrentSnapshot(ctx, true)
		_, err = c.finishTask(ctx, task, err)
		if err != nil {
			return errors.Wrapf(err, "While reverting to the most recent snapshot")
		}

		return nil
	}()

	if err != nil {
		switch err := errors.Cause(err).(type) {
		case *TimeoutExceededError:
			// handle specifically
			return fmt.Errorf("Timeout while attempting to revert to the most recent snapshot for a VM")
		default:
			// unknown error
			return errors.Wrap(err, "Got error while reverting to the most recent snapshot for a VM")
		}
	}

	return nil
}

// SnapshotRevertTo will revert the provided VM to the specified snapshot
func (c *Client) SnapshotRevertTo(vm *VirtualMachine, moRef *types.ManagedObjectReference) error {
	err := func() error {
		ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
		defer cancelFn()

		suppress := true
		req := types.RevertToSnapshot_Task{
			This:            *moRef,
			SuppressPowerOn: &suppress,
		}

		res, err := methods.RevertToSnapshot_Task(ctx, c.Client.Client, &req)
		err = c.checkErr(ctx, err)
		if err != nil {
			return err
		}

		task := object.NewTask(c.Client.Client, res.Returnval)
		_, err = c.finishTask(ctx, task, err)
		if err != nil {
			return errors.Wrapf(err, "While reverting to the most recent snapshot")
		}

		return nil
	}()

	if err != nil {
		switch err := errors.Cause(err).(type) {
		case *TimeoutExceededError:
			// handle specifically
			return fmt.Errorf("Timeout while attempting to revert to a specific snapshot for a VM")
		default:
			// unknown error
			return errors.Wrap(err, "Got error while reverting to a specific snapshot for a VM")
		}
	}

	return nil
}
