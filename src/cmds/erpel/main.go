package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// RootCmd is the base command when no other command has been specified.
var RootCmd = &cobra.Command{
	Use:   "erpel",
	Short: "filter log messages",
	Long: `
erpel is a program which filters log files for unwanted log messages and
prints the remaining messages. It detects whether or not a log message should
be ignored by applying a list of patterns, effectively "blacklisting" known log
messages that can safely be ignored.
`,
	SilenceErrors:     true,
	SilenceUsage:      true,
	PersistentPreRunE: parseConfig,
}

func main() {
	if cmd, err := RootCmd.ExecuteC(); err != nil {
		fmt.Printf("error: %v\n\n", err)
		cmd.Usage()
		os.Exit(1)
	}
}
