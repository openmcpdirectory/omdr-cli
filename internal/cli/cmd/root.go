package cmd

import (
	"fmt"
	"os"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/branding"
	clilogger "github.com/openmcpdirectory/omdr-cli/internal/cli/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	verbose    bool
	noBanner   bool
	jsonOutput bool
)

var rootCmd = &cobra.Command{
	Use:   "omdr",
	Short: "OMDR - Open MCP Directory CLI",
	Long: `Discover, install, and manage MCP servers from the Open MCP Directory.

  Website:  https://openmcpdirectory.com
  Docs:     https://docs.omdr.dev
  Registry: https://registry.omdr.dev`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		clilogger.SetVerbose(verbose)
		if !noBanner && !jsonOutput && cmd.Name() != "version" && cmd.Name() != "help" {
			branding.ShowBanner()
		}
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.omdr/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&noBanner, "no-banner", false, "disable banner display")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		viper.AddConfigPath(home + "/.omdr")
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("OMDR")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintln(os.Stderr, "error reading config:", err)
		}
	}
}
