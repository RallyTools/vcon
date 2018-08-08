# vcon

`vcon` (short for "VM Control") is a CLI tool to help manage VMs in a vSphere environment

## About

`vcon` is designed to help users script actions against a vSphere system, so that they may clone, power-cycle, and destroy VMs in an automated fashion.

## Uses

`vcon` is designed with the following tasks in mind.

### Connection testing

The `test` command will attempt to make a connection to vSphere without performing any actions.  This can help validate your credentials.

### Cloning

Using the `clone` command, `vcon` can duplicate a template or VM, ensure that it is in a running state, and provide information to find and access the new VM.  The user must specify the destination folder using the `--destination` flag.  The user _may_ optionally a name using the `--name` flag, but a name will be generated if none was provided.

The `--configuration` flag can be provided to reconfigure the VM after it's been cloned and before it is started.  The value is a JSON-formatted string; see [Configuration](#Configuration) for details.  Unlike the `configure` command, the string will not be read from STDIN when closing.

The `--on` flag can be set to `false` to prevent the VM from starting automatically.

### Info

The `info` command gets information about an existing VM, in JSON form.  The command indicates:

* The machine's configuration: number of CPUs, memory size (in MB), and network name
* The IPv4 address or addresses
* Whether the VM is currently running
* The path to the VM in it's data center
* The Managed Object Reference in vSphere

An example result:

``` json
{
  "configuration": {
    "cpus": 2,
    "memory": 12288,
    "network": "VLAN3028"
  },
  "ips": [],
  "isRunning": false,
  "path": "/Engineering/TeamSharks/temporary VMs/bob - 2018-05-09 14:47:59",
  "ref": "vm-1127"
}
```

### Configuration

The `configure` command allows the user to change certain virtual hardware allocations.  In particular, the CPU, memory, and network adapter may be changed.  The VM _must_ be powered off when making changes.

The configuration is provided as a JSON object:

``` json
{
	"cpus": number,
	"memory": number,
	"network": string
}
```

Only VM specifications that are in the provided JSON will be altered.  For example, to change the number of CPUs, but leave the memory and network unchanged, only include the `cpus` property.

When changing the network, it is assumed that all network devices will change to the requested network.

The JSON may be provided as an argument to the command, after the target VM, or read in from STDIN.  If the JSON is not read from stdid, then the argument may either be the literal JSON, or a path to a file containing the JSON:

``` sh
echo '{ "cpus": 2 }' | vcon configure $TARGET
vcon configure $TARGET "{ \"memory\": 2048 }"
vcon configure $TARGET /tmp/machine.json
```

### Annotation

Using the `note` command, `vcon` can append a new piece of text to a VM in vSphere.  The `--overwrite` flag can be used to replace any existing notes.

### Power cycling

Using the `power` command, `vcon` can turn on, turn off, or suspend a VM.

### Snapshoting _(experimental)_

The `snapshot` command will manage snapshots.  There are several subcommands: `create`, `list`, `remove`, and `revert`.  This functionality is not completely tested, and may change.

### Destroying

Using the `destroy` command, `vcon` can remove a VM from vSphere.  This will fail if the VM is currently running, but the command can stop the VM first by using the `--force` flag.

### Version

The `version` command returns the version of the binary.

## Vcon configuration

`vcon` needs several parameters in order for it to connect to, and work with, vSphere.  These parameters vary by command, but at the very least include user name, password, vSphere address, data center name, and data store name.  Parameters can be provided through a number of different mechanisms, including the command line, environment variables, and a config file.

### Command line options

Command line options are specified with a long name (i.e., `--username`), and when possible, a short name (i.e., `-u`).

### Environment variables

All environment variables are upper-cased versions of the command line option, and prefixed are with `VCON_`.  For example, the environment variable for `username` is `VCON_USERNAME`.  Most command line options may be specified by an environment variable.

### Config file

A configuration file, named `.vcon.[json|yaml]`, may be used to keep configuration.  By default, this file would be found in the current user's home directory, and the location may be specified using the `config` command line option.

### Precedence

The `vcon` CLI uses [Cobra](https://github.com/spf13/cobra) and [Viper](https://github.com/spf13/viper) to manage configuration, so as per the [precedence rules](https://github.com/spf13/viper#why-viper), the highest precedence is given to command line options, then environment variables, then configuration file settings, and finally default values.

### All options

| Option | Short name | Command | Env | Config file | Required | Default
|:--- |:--- |:--- |:---:|:---:|:---:|:---:|
| username | u | (all) | Y | Y | Y | |
| password | p | (all) | Y | Y | Y | |
| prompt-for-password | | (all) | | Y | | `true` |
| vsphere | v | (all) | Y | Y | Y |
| datacenter | | (all) | Y | Y | Y |
| datastore | | (all) | Y | Y | Y |
| timeout | t | (all) | Y | Y | | `30` |
| verbose | v | (all) |  | Y | | `false` |
| config | | (all) | | | | `~/.vcon.[json\|yaml]` |
| configuration | c | clone | | | |
| destination | d | clone | | Y | |
| name | n | clone, snapsnot-create | | | | (generated) |
| on | | clone | | | | `true` |
| resourcepool | | clone | Y | Y | Y | |
| force | f | destroy | | | | `false` |
| overwrite | | note | | | | `false` |
| snapshotIsRef| | snapshot-remove, snapshot-revert | | | | `false` |
| targetIsRef | | configure, destroy, info, note, power, snapshot-* | | | | `false` |

## Limitations

`vcon` is designed to _strictly_ operate within a single data center and data store.  If your requirements involve cloning virtual machines from one data store or data center to another, `vcon` is insufficient.

`vcon` cannot create _new_ VMs; it can only clone existing VMs and templates.

`vcon` is not designed for extensive VM alterations.  The CPU and memory can be changed, and the attached network may be changed.  If there are multiple network adapters, it is assumed that all network adapters will be changed to the same network.  Other VM features such as sound device, optical drive, and disk configuration cannot be changed with this tool.

## Examples

``` sh
#!/bin/bash
set -euo pipefail

# Testing the connection
vcon test

# Clone a template; don't spin up yet
RESULTS=$(vcon clone "/Engineering/templates/Deployment Template" --destination "/Engineering/TeamSharks/temporary VMs" --on=false)

# RESULTS is a JSON block, shaped like...
# {
#   "configuration": {
#     "cpus": 2,
#     "memory": 4096,
#     "network": "VLAN3000"
#   },
# 	"ips": [],
# 	"isRunning": false,
# 	"path": "/Engineering/TeamSharks/temporary VMs/MyName - 2018/04/29 20:44:22"
# 	"ref": "vm-139
# }

# Pull the id out of the result JSON
TARGET=$(echo $RESULTS | jq -r ".ref")

# Append some notes
vcon note $TARGET "VM generated for Team Shark acceptance test, feature F1234" --targetIsRef

# Change the VM's hardware configuration using HEREDOC
vcon configure $TARGET <<EOF
{
	"cpus": 6,
	"memory": 4096
}
EOF

# Power up VM
vcon power up $TARGET --targetIsRef

# Get the IP address
IP=$(vcon info $TARGET --targetIsRef | jq -r ".ips | .[0]")

# Execute some automated tests
$(REMOTE_IP=$IP testcafe ...)
SUCCESS=$?

if [[ $SUCCESS == 0 ]];
	# Power destroy the VM
	vcon destroy $TARGET --targetIsRef --force
else
	vcon note $TARGET "Test failed" --targetIsRef
fi
```