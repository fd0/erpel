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

var stateDir string

func init() {
	RootCmd.AddCommand(processCmd)
	processCmd.PersistentFlags().StringVarP(&stateDir, "state-dir", "s", "/var/lib/erpel", "set the directory for keeping log file positions")
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

		last, err := loadMarker(logfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading marker for %v: %v\n", logfile, err)
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

		if err = saveMarker(logfile, pos); err != nil {
			fmt.Fprintf(os.Stderr, "error saving marker for %v: %v\n", logfile, err)
		}
	}

	return nil
}
