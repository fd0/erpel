package main

import "fmt"

var version = "compiled manually"
var compiledAt = "unknown time"

var (
	verboseOutput bool
	debugOutput   bool
)

func init() {
	RootCmd.PersistentFlags().BoolVarP(&verboseOutput,
		"verbose", "v", false, "be verbose")
	RootCmd.PersistentFlags().BoolVar(&debugOutput,
		"debug", false, "be verbose")
}

// V prints the message when verbose is active.
func V(format string, args ...interface{}) {
	if !verboseOutput {
		return
	}

	fmt.Printf(format, args...)
}

// D prints the message when debug is active.
func D(format string, args ...interface{}) {
	if !debugOutput {
		return
	}

	fmt.Printf(format, args...)
}
