package main

import (
	"encoding/json"
	"erpel"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var processCmd = &cobra.Command{
	Use:     "process",
	Short:   "Process log files",
	Example: "$ erpel process /var/log/messages",
	Long: `
The process command is the main operation of erpel. It processes all logfile
specified on the command line, going throuh each file line by line and only
prints those log messages that do not match any of the process rules.
`,
	RunE: Process,
	PreRunE: func(*cobra.Command, []string) error {
		return LoadRules()
	},
}

var (
	stateDir      string
	rulesDir      string
	ignoreState   bool
	noUpdateState bool
)

func init() {
	RootCmd.AddCommand(processCmd)
	flags := processCmd.PersistentFlags()

	flags.StringVarP(&stateDir, "state-dir", "s", "/var/lib/erpel", "set the directory for keeping log file positions")
	bindConfigValue("state_dir", flags.Lookup("state-dir"))

	flags.StringVarP(&rulesDir, "rules", "r", "/etc/erpel/rules.d", "load rules from this directory")
	bindConfigValue("rules_dir", flags.Lookup("rules"))

	flags.BoolVarP(&ignoreState, "ignore-state", "i", false, "ignore the state and process the files from the start")
	flags.BoolVarP(&noUpdateState, "no-update-state", "n", false, "do not update the state")
}

func stateFilename(logfile string) string {
	base := strings.Replace(logfile, string(os.PathSeparator), ".", -1)
	return base + ".pos"
}

func loadMarker(logfile string) (m erpel.Marker, err error) {
	stateFile := filepath.Join(stateDir, stateFilename(logfile))

	D("trying to load position from state file %v\n", stateFile)

	f, err := os.Open(stateFile)
	if os.IsNotExist(err) {
		V("last position for %v not found\n", logfile)
		return m, nil
	}

	if err != nil {
		return m, nil
	}

	dec := json.NewDecoder(f)
	if err = dec.Decode(&m); err != nil {
		return m, err
	}

	if err = f.Close(); err != nil {
		return m, err
	}

	return m, nil
}

func saveMarker(logfile string, m erpel.Marker) error {
	stateFile := filepath.Join(stateDir, stateFilename(logfile))
	D("saving position to state file %v\n", stateFile)

	f, err := os.Create(stateFile)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(f)
	if err = enc.Encode(m); err != nil {
		return err
	}

	if err = f.Close(); err != nil {
		return err
	}

	return nil
}

// Process is the main command.
func Process(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("no log files to process")
	}

	for _, logfile := range args {
		V("processing log file %v\n", logfile)

		var (
			last erpel.Marker
			err  error
		)

		if !ignoreState {
			last, err = loadMarker(logfile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error loading marker for %v: %v\n", logfile, err)
			}
		}

		pos, err := erpel.ProcessFile(Rules, logfile, last, func(lines []string) error {
			for _, line := range lines {
				fmt.Println(line)
			}

			return nil
		})
		if err != nil {
			return err
		}

		if !noUpdateState {
			if err = saveMarker(logfile, pos); err != nil {
				fmt.Fprintf(os.Stderr, "error saving marker for %v: %v\n", logfile, err)
			}
		}
	}

	return nil
}
