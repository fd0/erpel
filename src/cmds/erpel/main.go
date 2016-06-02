package main

import (
	"erpel"
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
)

var opts = &struct {
	Verbose bool   `short:"v" long:"verbose" description:"be verbose"`
	Config  string `short:"c" long:"config" env:"ERPEL_CONFIG" default:"/etc/erpel/erpel.conf" description:"configuration file"`
}{}

// V prints the message when verbose is active.
func V(format string, args ...interface{}) {
	if !opts.Verbose {
		return
	}

	fmt.Printf(format, args...)
}

// E prints an error to stderr.
func E(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}

// Er prints the error err if it is set.
func Er(err error) {
	if err == nil {
		return
	}

	E("error: %v\n", err)
}

// Erx prints the error and exits with the given code, but only if the error is non-nil.
func Erx(err error, exitcode int) {
	if err == nil {
		return
	}

	Er(err)
	os.Exit(exitcode)
}

func main() {
	var parser = flags.NewParser(opts, flags.Default)

	_, err := parser.Parse()
	if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
		os.Exit(0)
	}
	Erx(err, 1)

	cfg, err := erpel.LoadConfig(opts.Config)
	if err != nil {
		Erx(err, 2)
	}

	fmt.Printf("cfg: %v\n", cfg)

	rules, err := erpel.LoadAllRules(cfg.RulesDir)
	if err != nil {
		Erx(err, 3)
	}

	fmt.Printf("loaded %v rules\n", len(rules))
}
