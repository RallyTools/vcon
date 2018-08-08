package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/RallyTools/vcon"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"
)

// Keys for configuration data
const (
	configurationKey     = "configuration"
	datacenterKey        = "datacenter"
	datastoreKey         = "datastore"
	destinationKey       = "destination"
	forceKey             = "force"
	nameKey              = "name"
	passwordKey          = "password"
	promptForPasswordKey = "prompt-for-password"
	resourcePoolKey      = "resourcepool"
	timeoutKey           = "timeout"
	usernameKey          = "username"
	verboseKey           = "verbose"
	vSphereKey           = "vsphere"
)

// ClientCommand embeds a cobra.Command and keeps a vcon.Client
type ClientCommand struct {
	cobra.Command
	c *vcon.Client
}

// NewClientCommand creates a new ClientCommand and assigns the PreRunE on
// the embedded cobra.Command to connect to the vSphere instance
func NewClientCommand(use, shortDescription string) *ClientCommand {
	cc := ClientCommand{
		Command: cobra.Command{
			Use:   use,
			Short: shortDescription,
		},
		c: nil,
	}

	cc.Command.PreRunE = cc.preRunE

	return &cc
}

func (cc *ClientCommand) preRunE(_ *cobra.Command, _ []string) error {
	password := viper.GetString(passwordKey)
	if len(password) == 0 && viper.GetBool(promptForPasswordKey) {
		// Password is not provided; request it now.
		fmt.Printf("Password for %s: ", viper.GetString(usernameKey))
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Printf("\n")
		if err != nil {
			return err
		}
		password = string(bytePassword)
		fmt.Printf("Proceeding...\n")
	}

	url := viper.GetString(vSphereKey)
	name := viper.GetString(usernameKey)
	datacenter := viper.GetString(datacenterKey)
	datastore := viper.GetString(datastoreKey)
	timeout := viper.GetInt(timeoutKey)
	c, err := vcon.NewClient(url, name, password, datacenter, datastore, timeout)
	if err != nil {
		return &vcon.ConnectionError{}
	}
	c.Verbose = viper.GetBool(verboseKey)
	cc.c = c

	return nil
}

// generateVMName returns a new name for a VM.  This may be specified by the
// `--name` flag to the `clone` verb, or if none is provided, one will be
// created using the current time and user name
func (cc *ClientCommand) generateVMName(providedName string) string {
	if providedName != "" {
		return providedName
	}

	now := time.Now()
	username := viper.GetString(usernameKey)
	atIndex := strings.Index(username, "@")
	if atIndex != -1 {
		username = username[:atIndex]
	}

	return fmt.Sprintf("%s - %04d-%02d-%02d %02d:%02d:%02d", username, now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
}

func (cc *ClientCommand) generateSnapshotName(providedName string) string {
	if providedName != "" {
		return providedName
	}

	now := time.Now()
	username := viper.GetString(usernameKey)
	atIndex := strings.Index(username, "@")
	if atIndex != -1 {
		username = username[:atIndex]
	}

	return fmt.Sprintf("Snapshot - %s - %04d-%02d-%02d %02d:%02d:%02d", username, now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
}

func (cc *ClientCommand) readString(params []string) (string, error) {
	// Get a reader; either Stdin or a specified path
	var r io.Reader
	if len(params) == 0 {
		r = os.Stdin
	} else {
		sourcePath := params[0]
		f, err := os.OpenFile(sourcePath, os.O_RDONLY, 0)
		if err != nil {
			if _, ok := err.(*os.PathError); !ok {
				return "", errors.Wrapf(err, "Got non-PathError err")
			}
		} else {
			r = f
		}
	}

	var value string
	if r != nil {
		// Have a reader; pull until EOF
		var b bytes.Buffer
		_, err := b.ReadFrom(r)
		if err != nil {
			return "", errors.Wrapf(err, "Nope...")
		}
		value = b.String()
	} else {
		// No reader; treat the argument as the note contents
		value = params[0]
	}

	return value, nil
}

func (cc *ClientCommand) writeSnapshotToConsole(snapshot *vcon.Snapshot) error {
	bytes, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to serialize to JSON")
	}

	os.Stdout.Write(bytes)
	os.Stdout.WriteString("\n")

	return nil
}

func (cc *ClientCommand) writeVMInfoToConsole(vm *vcon.VirtualMachine) error {
	vmi := cc.c.ReportVM(vm)
	bytes, err := json.MarshalIndent(vmi, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to serialize to JSON")
	}

	os.Stdout.Write(bytes)
	os.Stdout.WriteString("\n")

	return nil
}
