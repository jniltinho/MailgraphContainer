// Package cmd implements the mailgraph CLI using Cobra and Viper.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"mailgraph/internal/config"
)

var (
	rootCmd = &cobra.Command{
		Use:   "mailgraph",
		Short: "Mail statistics grapher for Postfix",
		Long:  "RRDtool frontend for mail statistics with interactive charts.",
	}

	cfgFile      string
	mailgraphCSS []byte
)

// SetMailgraphCSS provides embedded static assets from main before Execute.
func SetMailgraphCSS(css []byte) {
	mailgraphCSS = css
}

// Execute runs the root command and exits with status 1 on failure.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initConfig() {
	config.SetDefaults()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/mailgraph")
		viper.AddConfigPath("$HOME/.mailgraph")
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
	}

	viper.SetEnvPrefix("MAILGRAPH")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	_ = viper.ReadInConfig()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ./config.toml, /etc/mailgraph/config.toml)")
}