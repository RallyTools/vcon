package vcon

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

// Version is the version of the application and API
const Version = "0.3.0"

// PowerState describes whether a VM is on, off, or suspended
type PowerState string

const (
	// PoweredOff indicates that the VM is turned off
	PoweredOff PowerState = "powered_off"

	// PoweredOn indicates that the VM is running
	PoweredOn PowerState = "powered_on"

	// Suspended indicates that the VM is on but not running
	Suspended PowerState = "suspended"

	// Unknown indicates that the power state was not determined
	Unknown PowerState = "unknown"
)

// Client represents a connection to vSphere
type Client struct {
	Client *govmomi.Client
	Finder *find.Finder

	datacenter *object.Datacenter
	datastore  *object.Datastore
	timeout    time.Duration

	Verbose bool
}

// NewClient creates a connection to a vSphere instance
func NewClient(url, username, password, datacenter, datastore string, timeout int) (*Client, error) {
	connectionURL, err := buildConnectionString(url, username, password)
	if err != nil {
		return nil, err
	}

	c := &Client{
		timeout: time.Duration(timeout) * time.Second,
	}
	err = func() error {
		ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
		defer cancelFn()

		// Connect and log in to ESX or vCenter
		c.Client, err = govmomi.NewClient(ctx, connectionURL, true)
		if err := c.checkErr(ctx, err); err != nil {
			return errors.Wrapf(err, "Failed to connect to vSphere at '%s' with user '%s'", url, username)
		}

		c.Finder = find.NewFinder(c.Client.Client, false)
		c.datacenter, err = c.Finder.Datacenter(ctx, datacenter)
		if err := c.checkErr(ctx, err); err != nil {
			return errors.Wrapf(err, "Failed to find data center with name '%s'", datacenter)
		}

		c.Finder.SetDatacenter(c.datacenter)

		c.datastore, err = c.Finder.Datastore(ctx, datastore)
		if err := c.checkErr(ctx, err); err != nil {
			return errors.Wrapf(err, "Failed to find data store with name '%s'", datastore)
		}

		return nil
	}()

	if err != nil {
		switch err := errors.Cause(err).(type) {
		case *TimeoutExceededError:
			// handle specifically
			return nil, fmt.Errorf("Timeout while attempting to establish connection to vSphere")
		default:
			// unknown error
			return nil, errors.Wrap(err, "Got error while attempting to establish connection to vSphere")
		}
	}

	return c, nil
}

// AssignNote adds a note to the VM, or overwrites the notes entirely
func (c *Client) AssignNote(vm *VirtualMachine, note string, overwrite bool) error {
	if c.Verbose {
		fmt.Printf("Assigning note to VM...\n")
	}

	err := func() error {
		ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
		defer cancelFn()

		if !overwrite {
			originalNote := vm.MO.Config.Annotation
			if len(originalNote) != 0 {
				note = fmt.Sprintf("%s\n\n%s", originalNote, note)
			}
		}

		config := types.VirtualMachineConfigSpec{
			Annotation: note,
		}
		task, err := vm.VM.Reconfigure(ctx, config)
		_, err = c.finishTask(ctx, task, err)
		if err != nil {
			return errors.Wrapf(err, "Error while assigning note")
		}

		return nil
	}()

	if err != nil {
		switch err := errors.Cause(err).(type) {
		case *TimeoutExceededError:
			// handle specifically
			return fmt.Errorf("Timeout while attempting to assign note to VM")
		default:
			// unknown error
			return errors.Wrap(err, "Got error while assigning note to  VM")
		}
	}

	return nil
}

// Clone clones the specified VM
func (c *Client) Clone(vm *VirtualMachine, name, destination, resourcePool string) (*VirtualMachine, error) {
	if c.Verbose {
		fmt.Printf("Cloning VM...\n")
	}

	var newVM *object.VirtualMachine
	err := func() error {
		ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
		defer cancelFn()

		objPool, err := c.Finder.ResourcePoolOrDefault(ctx, resourcePool)
		if err := c.checkErr(ctx, err); err != nil {
			return errors.Wrapf(err, "While getting resource pool named '%s'", resourcePool)
		}

		// makeInventoryPath transforms a path to an inventory path by prepending
		inventoryDestination := c.makeInventoryPath(destination)
		objFolder, err := c.Finder.Folder(ctx, inventoryDestination)
		if err := c.checkErr(ctx, err); err != nil {
			return errors.Wrapf(err, "While getting folder named '%s'", destination)
		}

		objDsRef := c.datastore.Reference()
		objFolderRef := objFolder.Reference()
		objPoolRef := objPool.Reference()
		config := types.VirtualMachineCloneSpec{
			Location: types.VirtualMachineRelocateSpec{
				Datastore: &objDsRef,
				Folder:    &objFolderRef,
				Pool:      &objPoolRef,
			},
			Template: false,
		}

		task, err := vm.VM.Clone(ctx, objFolder, name, config)
		res, err := c.finishTask(ctx, task, err)
		if err != nil {
			return errors.Wrapf(err, "Error while cloning")
		}

		newVM = object.NewVirtualMachine(c.Client.Client, res.(types.ManagedObjectReference))
		return nil
	}()

	if err != nil {
		switch err := errors.Cause(err).(type) {
		case *TimeoutExceededError:
			// handle specifically
			return nil, fmt.Errorf("Timeout while attempting to clone VM")
		default:
			// unknown error
			return nil, errors.Wrap(err, "Got error while cloning a VM")
		}
	}

	result := &VirtualMachine{
		Ref: newVM.Reference(),
		VM:  newVM,
	}

	return result, nil
}

// Configure will change some of the virtual hardware that the specified VM
// uses
func (c *Client) Configure(vm *VirtualMachine, vmc *VirtualMachineConfiguration) error {
	if c.Verbose {
		fmt.Printf("Configuring VM...\n")
	}

	err := func() error {
		ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
		defer cancelFn()

		cspec := types.VirtualMachineConfigSpec{}
		reconfigure := false
		if vmc.CPUs != nil {
			cspec.NumCPUs = int32(*vmc.CPUs)
			reconfigure = true
		}
		if vmc.Memory != nil {
			cspec.MemoryMB = int64(*vmc.Memory)
			reconfigure = true
		}

		if reconfigure == true {
			task, err := vm.VM.Reconfigure(ctx, cspec)
			_, err = c.finishTask(ctx, task, err)
			if err = c.checkErr(ctx, err); err != nil {
				return err
			}
		}

		if vmc.Network != nil {
			devices, err := vm.VM.Device(ctx)
			if err = c.checkErr(ctx, err); err != nil {
				return err
			}

			dest := []mo.Network{}
			pc := property.DefaultCollector(c.Client.Client)
			err = pc.Retrieve(ctx, vm.MO.Network, []string{"name"}, &dest)
			if err = c.checkErr(ctx, err); err != nil {
				return err
			}

			network := dest[0]

			if network.Name == *vmc.Network {
				// There's nothing to change; exit early.
				return nil
			}

			backing := &types.VirtualEthernetCardNetworkBackingInfo{
				VirtualDeviceDeviceBackingInfo: types.VirtualDeviceDeviceBackingInfo{
					DeviceName: network.Name,
				},
			}
			matchingDevices := devices.SelectByBackingInfo(backing)

			requestedNetwork, err := c.Finder.Network(ctx, *vmc.Network)
			if err = c.checkErr(ctx, err); err != nil {
				return err
			}

			if requestedNetwork == nil {
				return fmt.Errorf("Failed to find requested network")
			}

			requestedBacking, err := requestedNetwork.EthernetCardBackingInfo(ctx)
			if err = c.checkErr(ctx, err); err != nil {
				return err
			}

			matchingDevices.Select(func(device types.BaseVirtualDevice) bool {
				device.GetVirtualDevice().Backing = requestedBacking
				err = vm.VM.EditDevice(ctx, device)
				if err = c.checkErr(ctx, err); err != nil {
					// Unclear what to do here; we will continue looping regardless.
				}

				// We are not collecting the results, so return false
				return false
			})
		}

		return nil
	}()

	if err != nil {
		switch err := errors.Cause(err).(type) {
		case *TimeoutExceededError:
			// handle specifically
			return fmt.Errorf("Timeout while attempting to reconfigure VM")
		default:
			// unknown error
			return errors.Wrap(err, "Got error while reconfiguring a VM")
		}
	}

	return nil
}

// Destroy will remove a VM from vSphere
func (c *Client) Destroy(vm *VirtualMachine) error {
	if c.Verbose {
		fmt.Printf("Destroying VM...\n")
	}

	err := func() error {
		ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
		defer cancelFn()

		powerState, err := vm.VM.PowerState(ctx)
		if err := c.checkErr(ctx, err); err != nil {
			return errors.Wrapf(err, "While getting getting power state")
		}

		if powerState != types.VirtualMachinePowerStatePoweredOff {
			return fmt.Errorf("Cannot destroy a Vm that is running")
		}

		task, err := vm.VM.Destroy(ctx)
		_, err = c.finishTask(ctx, task, err)
		if err != nil {
			return errors.Wrapf(err, "While destroying VM")
		}

		return nil
	}()

	if err != nil {
		switch err := errors.Cause(err).(type) {
		case *TimeoutExceededError:
			// handle specifically
			return fmt.Errorf("Timeout while attempting to destroy VM")
		default:
			// unknown error
			return errors.Wrap(err, "Got error while destroy a VM")
		}
	}

	return nil
}

// EnsureOff makes certain that the VM is off (not on or suspended)
func (c *Client) EnsureOff(vm *VirtualMachine) error {
	if c.Verbose {
		fmt.Printf("Ensuring off power state...\n")
	}

	err := func() error {
		ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
		defer cancelFn()

		task, err := vm.VM.PowerOff(ctx)
		_, err = c.finishTask(ctx, task, err)
		if err != nil {
			return errors.Wrapf(err, "While powering off")
		}

		return nil
	}()

	if err != nil {
		switch err := errors.Cause(err).(type) {
		case *TimeoutExceededError:
			// handle specifically
			return fmt.Errorf("Timeout while attempting to power off VM")
		default:
			// unknown error
			return errors.Wrap(err, "Got error while power off VM")
		}
	}

	return nil
}

// EnsureOn makes certain that the VM is on
func (c *Client) EnsureOn(vm *VirtualMachine) error {
	if c.Verbose {
		fmt.Printf("Ensuring on power state...\n")
	}

	err := func() error {
		ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
		defer cancelFn()

		task, err := vm.VM.PowerOn(ctx)
		_, err = c.finishTask(ctx, task, err)
		if err != nil {
			return errors.Wrapf(err, "While powering on")
		}

		_, err = vm.VM.WaitForIP(ctx)
		if err := c.checkErr(ctx, err); err != nil {
			return errors.Wrapf(err, "While waiting for IP address")
		}

		return nil
	}()

	if err != nil {
		switch err := errors.Cause(err).(type) {
		case *TimeoutExceededError:
			// handle specifically
			return fmt.Errorf("Timeout while attempting to power on VM")
		default:
			// unknown error
			return errors.Wrap(err, "Got error while power on VM")
		}
	}

	return nil
}

// GetPowerState returns the current power state of the provided VM
func (c *Client) GetPowerState(vm *VirtualMachine) (PowerState, error) {
	ps := Unknown
	err := func() error {
		ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
		defer cancelFn()

		s, err := vm.VM.PowerState(ctx)
		if err = c.checkErr(ctx, err); err != nil {
			return errors.Wrapf(err, "While getting getting power state")
		}

		switch s {
		case types.VirtualMachinePowerStatePoweredOff:
			ps = PoweredOff
		case types.VirtualMachinePowerStatePoweredOn:
			ps = PoweredOn
		case types.VirtualMachinePowerStateSuspended:
			ps = Suspended
		}

		return nil
	}()

	if err != nil {
		switch err := errors.Cause(err).(type) {
		case *TimeoutExceededError:
			// handle specifically
			return ps, fmt.Errorf("Timeout while getting power state of VM")
		default:
			// unknown error
			return ps, errors.Wrapf(err, "Got error while getting power state of VM")
		}
	}

	return ps, nil
}

func (c *Client) ReportSnapshot(mo *types.ManagedObjectReference) *Snapshot {
	s := &Snapshot{
		Ref: mo.Reference().Value,
	}

	return s
}

// ReportVM writes descriptive JSON data to the console
func (c *Client) ReportVM(vm *VirtualMachine) *VirtualMachineInfo {
	d := &VirtualMachineInfo{
		Configuration: &VirtualMachineConfiguration{},
	}

	func() {
		pc := property.DefaultCollector(c.Client.Client)

		ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
		defer cancelFn()

		d.Ref = vm.VM.Reference().Value

		powerState, err := vm.VM.PowerState(ctx)
		select {
		case <-ctx.Done():
			return
		default:
			if err != nil {
				return
			}
			d.IsRunning = powerState != types.VirtualMachinePowerStatePoweredOff
		}

		d.IPs = []string{}
		if d.IsRunning {
			macs, err := vm.VM.WaitForNetIP(ctx, true)
			select {
			case <-ctx.Done():
				return
			default:
				if err != nil {
					return
				}

				for _, addrs := range macs {
					d.IPs = append(d.IPs, addrs...)
				}
			}
		}

		elements, err := c.Finder.Element(ctx, vm.VM.Reference())
		select {
		case <-ctx.Done():
			return
		default:
			if err != nil {
				return
			}
			d.Path = c.makePath(elements.Path)
		}

		if vm.MO == nil {
			refs := []types.ManagedObjectReference{vm.VM.Reference()}
			res := []mo.VirtualMachine{}
			err = pc.Retrieve(ctx, refs, []string{"network", "summary"}, &res)
			select {
			case <-ctx.Done():
				return
			default:
				if err != nil {
					return
				}
				vm.MO = &res[0]
			}
		}

		cpuCount := int(vm.MO.Summary.Config.NumCpu)
		memorySize := int(vm.MO.Summary.Config.MemorySizeMB)
		d.Configuration.CPUs = &cpuCount
		d.Configuration.Memory = &memorySize

		dest := []interface{}{}
		err = pc.Retrieve(ctx, vm.MO.Network, []string{"name"}, &dest)
		err = c.checkErr(ctx, err)
		if err != nil {
			return
		}

		network := dest[0].(mo.Network)
		d.Configuration.Network = &network.Name

	}()

	return d
}

// Suspend makes certain that the VM is suspended
func (c *Client) Suspend(vm *VirtualMachine) error {
	if c.Verbose {
		fmt.Printf("Ensuring on suspended state...\n")
	}

	err := func() error {
		ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
		defer cancelFn()

		task, err := vm.VM.Suspend(ctx)
		_, err = c.finishTask(ctx, task, err)
		if err != nil {
			return errors.Wrapf(err, "While suspending")
		}

		return nil
	}()

	if err != nil {
		switch err := errors.Cause(err).(type) {
		case *TimeoutExceededError:
			// handle specifically
			return fmt.Errorf("Timeout while attempting to suspend VM")
		default:
			// unknown error
			return errors.Wrap(err, "Got error while attempting to suspend VM")
		}
	}

	return err
}

func buildConnectionString(vsphere, name, password string) (*url.URL, error) {
	if name == "" || password == "" {
		return nil, fmt.Errorf("Missing username or password")
	}

	connectionString := fmt.Sprintf("https://%s:%s@%s/sdk", name, password, vsphere)
	url, err := soap.ParseURL(connectionString)
	if err != nil {
		return nil, fmt.Errorf("Failed to form URL")
	}
	return url, nil
}

func (c *Client) checkErr(ctx context.Context, err error) error {
	select {
	case <-ctx.Done():
		return TimeoutExceededError{
			timeout: c.timeout,
		}
	default:
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) finishTask(ctx context.Context, task *object.Task, err error) (types.AnyType, error) {
	if err := c.checkErr(ctx, err); err != nil {
		return nil, errors.Wrapf(err, "While suspend")
	}

	ti, err := task.WaitForResult(ctx, nil)
	if err := c.checkErr(ctx, err); err != nil {
		return nil, errors.Wrapf(err, "While waiting for task to finish")
	}

	if ti.State != types.TaskInfoStateSuccess {
		return nil, fmt.Errorf(ti.Error.LocalizedMessage)
	}

	return ti.Result, nil
}

// makeInventoryPath transforms a path to an inventory path by prepending
// the datacenter name and "vm" path segments
func (c *Client) makeInventoryPath(path string) string {
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	datacenterName := c.datacenter.Name()
	completePath := fmt.Sprintf("/%s/vm/%s", datacenterName, path)
	return completePath
}

// makePath transforms an inventory path to a path by removing the datacenter
// name and "vm" path prefix segments
func (c *Client) makePath(inventoryPath string) string {
	path := inventoryPath
	datacenterNameSegment := fmt.Sprintf("/%s", c.datacenter.Name())
	if strings.HasPrefix(path, datacenterNameSegment) {
		path = path[len(datacenterNameSegment):]
		if strings.HasPrefix(path, "/vm/") {
			path = path[4:]
		}
	}
	return path
}
