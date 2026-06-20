package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/davidullrich/mailgraph/internal/config"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start collector and HTTP server for graphs",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			cmd.PrintErrln(err)
			os.Exit(1)
		}
		cfg.Serve = true
		cfg.Cat = false

		if err := runServer(cfg); err != nil {
			cmd.PrintErrln(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	bindConfigFlags(serverCmd)
}

func bindConfigFlags(cmd *cobra.Command) {
	cmd.Flags().String("logfile", "", "monitor logfile")
	cmd.Flags().String("logtype", "", "logfile type: syslog or metalog")
	cmd.Flags().Int("year", 0, "starting year of the log file")
	cmd.Flags().String("host", "", "use only entries for HOST (regexp) in syslog")
	cmd.Flags().String("daemon-rrd", "", "write RRDs to DIR")
	cmd.Flags().String("daemon-pid", "", "write PID to FILE")
	cmd.Flags().String("daemon-log", "", "write verbose-log to FILE")
	cmd.Flags().String("rrd-name", "", "use NAME.rrd and NAME_virus.rrd")
	cmd.Flags().Bool("ignore-localhost", false, "ignore mail to/from localhost")
	cmd.Flags().StringSlice("ignore-host", nil, "ignore mail to/from HOST regexp")
	cmd.Flags().Bool("only-mail-rrd", false, "update only the mail rrd")
	cmd.Flags().Bool("only-virus-rrd", false, "update only the virus rrd")
	cmd.Flags().Bool("rbl-is-spam", false, "count rbl rejects as spam")
	cmd.Flags().Bool("virbl-is-virus", false, "count virbl rejects as viruses")
	cmd.Flags().Bool("daemon", false, "write PID file and detach")
	cmd.Flags().Bool("verbose", false, "be verbose")
	cmd.Flags().String("listen", "", "HTTP listen address")
	cmd.Flags().String("hostname", "", "hostname shown in graph title")
	cmd.Flags().Bool("tls", false, "enable HTTPS with TLS certificate")
	cmd.Flags().String("tls-cert", "", "TLS certificate file (PEM)")
	cmd.Flags().String("tls-key", "", "TLS private key file (PEM)")
	cmd.Flags().Bool("auth", false, "enable HTTP Basic authentication")
	cmd.Flags().String("auth-user", "", "HTTP Basic auth username")
	cmd.Flags().String("auth-pass", "", "HTTP Basic auth password")
	cmd.Flags().String("auth-realm", "", "HTTP Basic auth realm (default Mailgraph)")

	_ = viper.BindPFlag("log.file", cmd.Flags().Lookup("logfile"))
	_ = viper.BindPFlag("log.type", cmd.Flags().Lookup("logtype"))
	_ = viper.BindPFlag("log.year", cmd.Flags().Lookup("year"))
	_ = viper.BindPFlag("log.host_filter", cmd.Flags().Lookup("host"))
	_ = viper.BindPFlag("rrd.dir", cmd.Flags().Lookup("daemon-rrd"))
	_ = viper.BindPFlag("daemon.pid_file", cmd.Flags().Lookup("daemon-pid"))
	_ = viper.BindPFlag("daemon.log_file", cmd.Flags().Lookup("daemon-log"))
	_ = viper.BindPFlag("rrd.name", cmd.Flags().Lookup("rrd-name"))
	_ = viper.BindPFlag("filter.ignore_localhost", cmd.Flags().Lookup("ignore-localhost"))
	_ = viper.BindPFlag("filter.ignore_hosts", cmd.Flags().Lookup("ignore-host"))
	_ = viper.BindPFlag("rrd.only_mail", cmd.Flags().Lookup("only-mail-rrd"))
	_ = viper.BindPFlag("rrd.only_virus", cmd.Flags().Lookup("only-virus-rrd"))
	_ = viper.BindPFlag("filter.rbl_is_spam", cmd.Flags().Lookup("rbl-is-spam"))
	_ = viper.BindPFlag("filter.virbl_is_virus", cmd.Flags().Lookup("virbl-is-virus"))
	_ = viper.BindPFlag("daemon.enabled", cmd.Flags().Lookup("daemon"))
	_ = viper.BindPFlag("app.verbose", cmd.Flags().Lookup("verbose"))
	_ = viper.BindPFlag("server.listen", cmd.Flags().Lookup("listen"))
	_ = viper.BindPFlag("server.hostname", cmd.Flags().Lookup("hostname"))
	_ = viper.BindPFlag("server.tls_enabled", cmd.Flags().Lookup("tls"))
	_ = viper.BindPFlag("server.tls_cert", cmd.Flags().Lookup("tls-cert"))
	_ = viper.BindPFlag("server.tls_key", cmd.Flags().Lookup("tls-key"))
	_ = viper.BindPFlag("auth.enabled", cmd.Flags().Lookup("auth"))
	_ = viper.BindPFlag("auth.username", cmd.Flags().Lookup("auth-user"))
	_ = viper.BindPFlag("auth.password", cmd.Flags().Lookup("auth-pass"))
	_ = viper.BindPFlag("auth.realm", cmd.Flags().Lookup("auth-realm"))
}