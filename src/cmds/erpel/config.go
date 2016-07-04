package main

import (
	"erpel"
	"fmt"

	"github.com/BurntSushi/xdg"
	"github.com/spf13/cobra"
)

var configFile string
var configPaths = xdg.Paths{}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file to read at startup (default is $XDG_CONFIG_FILE/erpel.conf)")
}

const configFileName = "erpel.conf"

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

	cfg, err := erpel.ParseConfigFile(configFile)
	if err != nil {
		return fmt.Errorf("parse config file %v failed: %v", configFile, err)
	}

	fmt.Printf("cfg: %v\n", cfg)

	return nil
}
