package main

import (
	"erpel"
	"fmt"

	"github.com/BurntSushi/xdg"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var configFile string
var configPaths = xdg.Paths{}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file to read at startup (default is $XDG_CONFIG_FILE/erpel.conf)")
}

const configFileName = "erpel.conf"

var cfg erpel.Config

// initConfig parses the configuration file.
func initConfig() {
	var err error
	if configFile == "" {
		configFile, err = configPaths.ConfigFile(configFileName)
		if err != nil {
			V("%v\n", err)
			return
		}
	}

	V("config file is %q\n", configFile)
}

func parseConfig(cmd *cobra.Command, args []string) error {
	if configFile == "" {
		return nil
	}

	V("load config file %q\n", configFile)

	c, err := erpel.ParseConfigFile(configFile)
	if err != nil {
		return fmt.Errorf("parse config file %v failed: %v", configFile, err)
	}

	cfg = c

	if cfg.RulesDir != "" {
		if f, ok := configBinds["rules_dir"]; ok {
			f.Value.Set(cfg.RulesDir)
		}
	}

	return nil
}

var configBinds map[string]*pflag.Flag

func bindConfigValue(name string, flag *pflag.Flag) {
	if configBinds == nil {
		configBinds = make(map[string]*pflag.Flag)
	}

	configBinds[name] = flag
}
