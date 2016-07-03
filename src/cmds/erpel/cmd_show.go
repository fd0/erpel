package main

import (
	"erpel"
	"errors"
	"io"
	"os"
	"text/template"

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
		return ShowRules(os.Stdout, args)
	},
}

func init() {
	RootCmd.AddCommand(showCmd)
}

const outputTemplate = `Rules from {{.Filename}}:

Templates:
{{range .Templates}}
{{- .}}
{{end -}}

`

var tmpl = template.Must(template.New("output").Parse(outputTemplate))

// ShowRules visualises an erpel rule file.
func ShowRules(wr io.Writer, args []string) error {
	if len(args) == 0 {
		return errors.New("no rule file specified, nothing to do")
	}

	if len(args) > 1 {
		return errors.New("more than one rule file specified")
	}

	filename := args[0]

	rules, err := erpel.ParseRulesFile(filename)
	if err != nil {
		return err
	}

	if err = rules.Check(); err != nil {
		return err
	}

	data := struct {
		Filename  string
		Templates []erpel.RuleView
	}{
		filename,
		rules.Views(),
	}

	tmpl.Execute(wr, data)

	return nil
}
