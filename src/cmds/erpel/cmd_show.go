package main

import (
	"erpel"
	"errors"
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:     "show [flags] rulefile",
	Example: "$ erpel show /etc/erpel/rules.d/dovecot",
	Short:   "Parse and show a rules file",
	Long: `
The show command parses and visualises a file containing erpel ignore rules.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return ShowRules(args)
	},
}

// show templates instead of field names
var displayTemplates bool

func init() {
	RootCmd.AddCommand(showCmd)

	showCmd.Flags().BoolVarP(&displayTemplates, "templates", "t", false, "show templates instead of field names")
}

var (
	printText        = color.New(color.FgWhite).PrintfFunc()
	printField       = color.New(color.FgHiRed).PrintfFunc()
	printGlobalField = color.New(color.FgHiBlue).PrintfFunc()
)

// ShowRules visualises an erpel rule file.
func ShowRules(args []string) error {
	if len(args) == 0 {
		return errors.New("no rule file specified, nothing to do")
	}

	if len(args) > 1 {
		return errors.New("more than one rule file specified")
	}

	filename := args[0]

	rules, err := erpel.ParseRulesFile(cfg.Fields, filename)
	if err != nil {
		return err
	}

	if err = rules.Check(); err != nil {
		return err
	}

	fmt.Printf("Rules from %v:\n", filename)
	for _, rv := range rules.Views() {
		for _, field := range rv {
			switch f := field.(type) {
			case erpel.Text:
				fmt.Printf("%s", f)
			case erpel.FieldView:
				p := printField
				if f.Global {
					p = printGlobalField
				}

				if displayTemplates {
					p("%s", f.S)
				} else {
					p("%s", f.F.Name)
				}
			}
		}
		fmt.Println()
	}

	return nil
}
