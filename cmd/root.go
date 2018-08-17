package cmd

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// InitializeCommands sets up the cobra commands
func InitializeCommands() *cobra.Command {
	rootCmd := createRootCommand()

	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(
		createCloneCommand(),
		createConfigureCommand(),
		createDestroyCommand(),
		createInfoCommand(),
		createInitCommand(),
		createNoteCommand(),
		createPowerCommand(),
		createRelocateCommand(),
		createSnapshotCommand(),
		createTestCommand(),
		createVersionCommand(),
	)

	return rootCmd
}

const longRootDescription = `vcon (short for "VM Control") performs vSphere management tasks

vcon makes requests to vSphere in groups, clustered together via a timeout`

func createRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vcon",
		Short: "vcon performs vSphere management tasks",
		Long:  longRootDescription,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}

	viper.SetEnvPrefix("VCON")

	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.vcon.yaml)")

	cmd.PersistentFlags().String(datacenterKey, "", "vSphere datacenter name")
	viper.BindEnv(datacenterKey)
	viper.BindPFlag(datacenterKey, cmd.PersistentFlags().Lookup(datacenterKey))

	cmd.PersistentFlags().String(datastoreKey, "", "vSphere datastore name")
	viper.BindEnv(datastoreKey)
	viper.BindPFlag(datastoreKey, cmd.PersistentFlags().Lookup(datastoreKey))

	cmd.PersistentFlags().Bool(promptForPasswordKey, true, "prompts for password when password is not provided")
	viper.BindEnv(promptForPasswordKey)
	viper.BindPFlag(promptForPasswordKey, cmd.PersistentFlags().Lookup(promptForPasswordKey))

	cmd.PersistentFlags().StringP(passwordKey, "p", "", "vSphere user password")
	viper.BindEnv(passwordKey)
	viper.BindPFlag(passwordKey, cmd.PersistentFlags().Lookup(passwordKey))

	cmd.PersistentFlags().IntP(timeoutKey, "t", 30, "timeout for operations, in seconds")
	viper.BindEnv(timeoutKey)
	viper.BindPFlag(timeoutKey, cmd.PersistentFlags().Lookup(timeoutKey))

	cmd.PersistentFlags().StringP(usernameKey, "u", "", "vSphere user name")
	viper.BindEnv(usernameKey)
	viper.BindPFlag(usernameKey, cmd.PersistentFlags().Lookup(usernameKey))

	cmd.PersistentFlags().BoolP(verboseKey, "v", false, "causes vcon to emit progress messages")
	viper.BindPFlag(verboseKey, cmd.PersistentFlags().Lookup(verboseKey))

	cmd.PersistentFlags().String(vSphereKey, "", "DNS name or IP address of the vSphere instance")
	viper.BindEnv(vSphereKey)
	viper.BindPFlag(vSphereKey, cmd.PersistentFlags().Lookup(vSphereKey))

	return cmd
}

func initConfig() {
	if cfgFile != "" {
		_, err := os.Stat(cfgFile)
		if err != nil {
			perr := err.(*os.PathError)
			fmt.Printf("Could not use requested config file: %s\n", perr.Error())
			cfgFile = ""
		} else {
			// Use config file from the flag.
			viper.SetConfigFile(cfgFile)
		}
	}

	if cfgFile == "" {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".vcon" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".vcon")
	}

	// It may be that the file doesn't exist; that's OK.
	viper.ReadInConfig()
}
