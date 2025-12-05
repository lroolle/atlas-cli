package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lroolle/atlas-cli/internal/version"
	"github.com/lroolle/atlas-cli/pkg/cmd/page"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:     "atl",
	Short:   "CLI for Atlassian REST API",
	Long:    `Atlas CLI provides command-line access to Atlassian Bitbucket, JIRA, and Confluence REST APIs`,
	Version: version.Full(),
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $XDG_CONFIG_HOME/atlas/config.yaml)")
	rootCmd.PersistentFlags().String("username", "", "Username for authentication")

	viper.BindPFlag("username", rootCmd.PersistentFlags().Lookup("username"))

	rootCmd.AddCommand(page.NewCmdPage())
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			configDir = filepath.Join(home, ".config")
		}

		atlasConfigDir := filepath.Join(configDir, "atlas")
		viper.AddConfigPath(atlasConfigDir)
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("ATLAS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
