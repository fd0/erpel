package main

import (
	"erpel"
	"errors"
	"io"
	"os"
	"text/template"

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
		return ShowRules(os.Stdout, args)
	},
}

// show templates instead of field names
var displayTemplates bool

func init() {
	RootCmd.AddCommand(showCmd)

	showCmd.Flags().BoolVarP(&displayTemplates, "templates", "t", false, "show templates instead of field names")
}

const outputTemplate = `Rules from {{.Filename}}:

Templates:
{{range .Templates}}
{{- .}}
{{end -}}

`

var (
	printText  = color.New(color.FgWhite).SprintfFunc()
	printField = color.New(color.FgGreen).SprintfFunc()
)

func colorPrinter(templates []erpel.RuleView) []string {
	list := make([]string, 0, len(templates))
	for _, template := range templates {
		var s string
		for _, field := range template {
			switch f := field.(type) {
			case erpel.Text:
				s += printText("%s", f)
			case erpel.FieldView:
				if displayTemplates {
					s += printField("%s", f.S)
				} else {
					s += printField("%s", f.F.Name)
				}
			}
		}
		list = append(list, s)
	}
	return list
}

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

	rules, err := erpel.ParseRulesFile(cfg.Fields, filename)
	if err != nil {
		return err
	}

	if err = rules.Check(); err != nil {
		return err
	}

	data := struct {
		Filename  string
		Templates []string
	}{
		filename,
		colorPrinter(rules.Views()),
	}

	tmpl.Execute(wr, data)

	return nil
}
