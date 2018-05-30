package vcon

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// VirtualMachine is a vSphere VM
type VirtualMachine struct {
	MO  *mo.VirtualMachine
	Ref types.ManagedObjectReference
	VM  *object.VirtualMachine
}

// VirtualMachineConfiguration describes the virtual hardware assigned to a VM
type VirtualMachineConfiguration struct {
	CPUs    *int    `json:"cpus,omitempty"`
	Memory  *int    `json:"memory,omitempty"`
	Network *string `json:"network,omitempty"`
}

// VirtualMachineInfo describes interesting information about a VM
type VirtualMachineInfo struct {
	Configuration *VirtualMachineConfiguration `json:"configuration"`
	IPs           []string                     `json:"ips"`
	IsRunning     bool                         `json:"isRunning"`
	Path          string                       `json:"path"`
	Ref           string                       `json:"ref"`
}

// FindVM will fetch the Virtual Machine struct for use with this API.  The VM
// can be identified by path (example, `/Engineering/Templates/Base Template`),
// or by the Managed Object Ref (example, `vm-139`).  The struct can also be
// populated with the additional data as requested in the `properties`
// argument.
func (c *Client) FindVM(path string, pathIsRef bool, properties ...string) (*VirtualMachine, error) {
	// Example complete inventory path
	// /Static/vm/Rally/Engineering/AC2GO/Templates/AC2Go RIO Studio Template (Built 2018-04-24)

	if c.Verbose {
		fmt.Printf("Finding VM at path: %s...\n", path)
	}

	var moVM *mo.VirtualMachine
	var ref types.ManagedObjectReference
	var vm *object.VirtualMachine
	err := func() error {
		ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
		defer cancelFn()

		if pathIsRef {
			ref = types.ManagedObjectReference{
				Type:  "VirtualMachine",
				Value: path,
			}
		} else {
			inventoryPath := c.makeInventoryPath(path)
			searchIndex := object.NewSearchIndex(c.Client.Client)
			inventoryRef, err := searchIndex.FindByInventoryPath(ctx, inventoryPath)
			if err := c.checkErr(ctx, err); err != nil {
				return errors.Wrap(err, "Failed to find VM by inventory path")
			}
			if inventoryRef == nil {
				return fmt.Errorf("Failed to find VM '%s'", path)
			}
			ref = inventoryRef.Reference()
		}
		vm = object.NewVirtualMachine(c.Client.Client, ref)

		if len(properties) != 0 {
			pc := property.DefaultCollector(c.Client.Client)
			res := []mo.VirtualMachine{}
			refs := []types.ManagedObjectReference{vm.Reference()}
			err := pc.Retrieve(ctx, refs, properties, &res)
			if err := c.checkErr(ctx, err); err != nil {
				return errors.Wrap(err, "While getting properties")
			}

			moVM = &res[0]
		}

		return nil
	}()

	if err != nil {
		switch err := errors.Cause(err).(type) {
		case *TimeoutExceededError:
			// handle specifically
			return nil, fmt.Errorf("Timeout while finding VM '%s'", path)
		default:
			// unknown error
			return nil, errors.Wrapf(err, "Got error while finding VM '%s'", path)
		}
	}

	result := &VirtualMachine{
		MO:  moVM,
		Ref: ref,
		VM:  vm,
	}

	return result, nil
}
