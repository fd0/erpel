package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long: `
Print detailed information about the build environment and the version of this
software.
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("erpel %s compiled with %v\n",
			version, runtime.Version())
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
