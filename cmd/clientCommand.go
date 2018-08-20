package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/user"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/RallyTools/vcon"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vjeantet/jodaTime"
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

const (
	defaultDateTimeFormat = "YYYY-MM-dd hh:mm:ss"
	snapshotNameTemplate  = "Snapshot - {{ Username }} - {{ Now }}"
	vmNameTemplate        = "{{ Username }} - {{ Now }}"
)

// ClientCommand embeds a cobra.Command and keeps a vcon.Client
type ClientCommand struct {
	cobra.Command
	c *vcon.Client

	nameTmpl *template.Template
}

// NewClientCommand creates a new ClientCommand and assigns the PreRunE on
// the embedded cobra.Command to connect to the vSphere instance
func NewClientCommand(use, shortDescription string) *ClientCommand {
	formatTime := func(now time.Time, format ...string) string {
		formatter := defaultDateTimeFormat
		if len(format) > 0 {
			formatter = format[0]
		}
		return jodaTime.Format(formatter, now)
	}
	trimAtsign := func(username string) string {
		atIndex := strings.Index(username, "@")
		if atIndex != -1 {
			username = username[:atIndex]
		}
		return username
	}

	fm := template.FuncMap{
		"Env": os.Getenv,
		"Now": func(format ...string) string {
			return formatTime(time.Now(), format...)
		},
		"Username": func() string {
			usr, err := user.Current()
			if err != nil {
				// Failed to get the current user
				return "Unknown user"
			}
			return trimAtsign(usr.Name)
		},
		"UtcNow": func(format ...string) string {
			return formatTime(time.Now().UTC(), format...)
		},
		"VsUsername": func() string {
			username := viper.GetString(usernameKey)
			return trimAtsign(username)
		},
	}

	tmpl := template.New("name_template").Funcs(fm)

	cc := ClientCommand{
		Command: cobra.Command{
			Use:   use,
			Short: shortDescription,
		},
		c:        nil,
		nameTmpl: tmpl,
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
func (cc *ClientCommand) generateVMName(nameTemplate string) string {
	if nameTemplate == "" {
		nameTemplate = vmNameTemplate
	}

	return cc.generateName(nameTemplate)
}

func (cc *ClientCommand) generateSnapshotName(nameTemplate string) string {
	if nameTemplate == "" {
		nameTemplate = snapshotNameTemplate
	}

	return cc.generateName(nameTemplate)
}

func (cc *ClientCommand) generateName(template string) string {
	tmpl, err := cc.nameTmpl.Parse(template)
	if err != nil {
		fmt.Printf("Template parsing error: %s\n", err.Error())
		uuid, _ := uuid.NewUUID()
		return fmt.Sprintf("%s", uuid.String())
	}

	var sb strings.Builder
	err = tmpl.Execute(&sb, nil)
	if err != nil {
		fmt.Printf("Template execution error: %s\n", err.Error())
		uuid, _ := uuid.NewUUID()
		return fmt.Sprintf("%s", uuid.String())
	}

	return sb.String()
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
