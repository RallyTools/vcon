package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
	"syscall"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	yaml "gopkg.in/yaml.v2"
)

func createInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Creates a '.vcon.[json|yaml]' configuration file in your home directory",
		RunE: func(_ *cobra.Command, _ []string) error {
			type config struct {
				Username   string `json:"username,omitempty" yaml:"username,omitempty"`
				Password   string `json:"password,omitempty" yaml:"password,omitempty"`
				VSphere    string `json:"vsphere,omitempty" yaml:"vsphere,omitempty"`
				Datacenter string `json:"datacenter,omitempty" yaml:"datacenter,omitempty"`
				Datastore  string `json:"datastore,omitempty" yaml:"datastore,omitempty"`
			}

			cfg := config{}

			fmt.Printf("Config file flag: '%s'\n", cfgFile)
			if cfgFile == "" {
				home, _ := homedir.Dir()
				cfgFile = path.Join(home, ".vcon.yaml")
			}
			f, err := os.OpenFile(cfgFile, os.O_CREATE&os.O_WRONLY, 0)
			if err != nil {
				return err
			}

			s := getValueFromUser("Please enter your username for vSphere", "Username")
			if s != "" {
				cfg.Username = s
			}

			if cfg.Username != "" {
				fmt.Printf("\n")
				fmt.Printf("Please enter the password for user '%s'\n", cfg.Username)
				fmt.Printf("NOTE: your username will be stored in PLAIN TEXT; leave blank for shared or insecure systems!\n")
				fmt.Printf("Password: ")
				bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
				fmt.Printf("\n")
				if err != nil {
					return err
				}
				cfg.Password = string(bytePassword)
			}

			s = getValueFromUser("Please enter the machine name or IP address of your vSphere installation", "Address")
			if s != "" {
				cfg.VSphere = s
			}

			s = getValueFromUser("Please enter the name of your vSphere data center", "Data center")
			if s != "" {
				cfg.Datacenter = s
			}

			s = getValueFromUser("Please enter the name of your vSphere data store", "Data store")
			if s != "" {
				cfg.Datastore = s
			}

			encoder := yaml.NewEncoder(f)
			err = encoder.Encode(cfg)
			if err != nil {
				return err
			}

			err = f.Close()
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func getValueFromUser(message, prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\n%s\n%s: ", message, prompt)
	str, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}
	str = strings.TrimSpace(str)
	return str
}
