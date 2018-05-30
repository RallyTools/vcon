package main

import (
	"fmt"
	"os"

	"github.com/RallyTools/vcon"
	"github.com/RallyTools/vcon/cmd"
)

func main() {
	rootCmd := cmd.InitializeCommands()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		if cerr, ok := err.(vcon.ErrorCoder); ok {
			os.Exit(cerr.Code())
		}
		os.Exit(-1)
	}
}
