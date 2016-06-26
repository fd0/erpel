package main

import (
	"bufio"
	"erpel"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/jessevdk/go-flags"
)

var version = "compiled manually"
var compiledAt = "unknown time"

var opts = &struct {
	Verbose  bool     `short:"v" long:"verbose" description:"be verbose"`
	Debug    bool     `          long:"debug" description:"turn on debugging"`
	Config   string   `short:"c" long:"config" env:"ERPEL_CONFIG" default:"/etc/erpel/erpel.conf" description:"configuration file"`
	Logfiles []string `short:"l" long:"logfile" description:"logfile to process"`

	Version bool `short:"V" long:"version" description:"print version information"`
}{}

// V prints the message when verbose is active.
func V(format string, args ...interface{}) {
	if !opts.Verbose {
		return
	}

	fmt.Printf(format, args...)
}

// D prints the message when debug is active.
func D(format string, args ...interface{}) {
	if !opts.Debug {
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

	if opts.Version {
		fmt.Printf("erpel version %v, compiled at %v using %v\n", version, compiledAt, runtime.Version())
		return
	}

	if len(opts.Logfiles) == 0 {
		E("no logfile specified, use --logfile\n")
		os.Exit(1)
	}

	cfg, err := erpel.ParseConfigFile(opts.Config)
	if err != nil {
		Erx(err, 2)
	}

	V("config loaded from %v\n", opts.Config)
	D("  config: %#v\n", cfg)

	rules, err := erpel.ParseAllRulesFiles(cfg.RulesDir)
	if err != nil {
		Erx(err, 3)
	}

	V("loaded %v rules from %v\n", len(rules), cfg.RulesDir)

	if opts.Debug {
		for _, rule := range rules {
			for _, r := range rule.RegExps() {
				D("  Rule: %v\n", r.String())
			}
		}
	}

	for _, logfile := range opts.Logfiles {
		V("processing %v\n", logfile)

		f, err := os.Open(logfile)
		if err != nil {
			E("error opening logfile %v: %v\n", logfile, err)
			continue
		}

		sc := bufio.NewScanner(f)
	nextLine:
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())

			for _, rule := range rules {
				if rule.Match(line) {
					continue nextLine
				}
			}

			fmt.Println(line)
		}

		err = f.Close()
		if err != nil {
			E("error closing logfile %v: %v\n", logfile, err)
			continue
		}
	}
}
