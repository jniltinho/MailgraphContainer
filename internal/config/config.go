package config

import (
	"flag"
	"fmt"
	"os"
	"time"
)

var (
	Version   = "2.0.0"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

type Config struct {
	LogFile         string
	LogType         string
	Year            int
	HostFilter      string
	RRDDir          string
	PIDFile         string
	DaemonLogFile   string
	RRDName         string
	IgnoreLocalhost bool
	IgnoreHosts     []string
	OnlyMailRRD     bool
	OnlyVirusRRD    bool
	RBLIsSpam       bool
	VirblIsVirus    bool
	Daemon          bool
	Cat             bool
	Verbose         bool
	Serve           bool
	ListenAddr      string
	Hostname        string
}

func Default() Config {
	hostname, _ := os.Hostname()
	return Config{
		LogFile:       "/var/log/mail/mail.log",
		LogType:       "syslog",
		Year:          time.Now().Year(),
		RRDDir:        "/var/www/mailgraph/rrd",
		PIDFile:       "/var/run/mailgraph.pid",
		DaemonLogFile: "/var/log/mailgraph.log",
		RRDName:       "mailgraph",
		ListenAddr:    ":8080",
		Hostname:      hostname,
		Serve:         true,
	}
}

func Parse() (Config, error) {
	cfg := Default()

	flag.StringVar(&cfg.LogFile, "logfile", cfg.LogFile, "monitor logfile instead of /var/log/syslog")
	flag.StringVar(&cfg.LogFile, "l", cfg.LogFile, "monitor logfile (short)")
	flag.StringVar(&cfg.LogType, "logtype", cfg.LogType, "logfile type: syslog or metalog")
	flag.StringVar(&cfg.LogType, "t", cfg.LogType, "logfile type (short)")
	flag.IntVar(&cfg.Year, "year", cfg.Year, "starting year of the log file")
	flag.IntVar(&cfg.Year, "y", cfg.Year, "starting year (short)")
	flag.StringVar(&cfg.HostFilter, "host", "", "use only entries for HOST (regexp) in syslog")
	flag.StringVar(&cfg.RRDDir, "daemon-rrd", cfg.RRDDir, "write RRDs to DIR")
	flag.StringVar(&cfg.PIDFile, "daemon-pid", cfg.PIDFile, "write PID to FILE")
	flag.StringVar(&cfg.DaemonLogFile, "daemon-log", cfg.DaemonLogFile, "write verbose-log to FILE")
	flag.StringVar(&cfg.RRDName, "rrd-name", cfg.RRDName, "use NAME.rrd and NAME_virus.rrd")
	flag.BoolVar(&cfg.IgnoreLocalhost, "ignore-localhost", false, "ignore mail to/from localhost")
	flag.BoolVar(&cfg.OnlyMailRRD, "only-mail-rrd", false, "update only the mail rrd")
	flag.BoolVar(&cfg.OnlyVirusRRD, "only-virus-rrd", false, "update only the virus rrd")
	flag.BoolVar(&cfg.RBLIsSpam, "rbl-is-spam", false, "count rbl rejects as spam")
	flag.BoolVar(&cfg.VirblIsVirus, "virbl-is-virus", false, "count virbl rejects as viruses")
	flag.BoolVar(&cfg.Daemon, "daemon", false, "start collector in the background")
	flag.BoolVar(&cfg.Daemon, "d", false, "start collector in background (short)")
	flag.BoolVar(&cfg.Cat, "cat", false, "read logfile once and exit")
	flag.BoolVar(&cfg.Cat, "c", false, "read logfile once (short)")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "be verbose")
	flag.BoolVar(&cfg.Verbose, "v", false, "be verbose (short)")
	flag.BoolVar(&cfg.Serve, "serve", true, "start HTTP server for graphs")
	flag.StringVar(&cfg.ListenAddr, "listen", cfg.ListenAddr, "HTTP listen address")
	flag.StringVar(&cfg.Hostname, "hostname", cfg.Hostname, "hostname shown in graph title")

	var help, version bool
	flag.BoolVar(&help, "help", false, "display help")
	flag.BoolVar(&help, "h", false, "display help (short)")
	flag.BoolVar(&version, "version", false, "output version")
	flag.BoolVar(&version, "V", false, "output version (short)")

	var ignoreHosts multiFlag
	flag.Var(&ignoreHosts, "ignore-host", "ignore mail to/from HOST regexp (repeatable)")

	flag.Parse()

	if help {
		printUsage()
		os.Exit(0)
	}
	if version {
		fmt.Printf("mailgraph %s (Go port) %s %s\n", Version, BuildDate, GitCommit)
		os.Exit(0)
	}

	cfg.IgnoreHosts = ignoreHosts

	if cfg.OnlyMailRRD && cfg.OnlyVirusRRD {
		return cfg, fmt.Errorf("cannot use --only-mail-rrd and --only-virus-rrd together")
	}

	return cfg, nil
}

type multiFlag []string

func (m *multiFlag) String() string { return "" }

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func printUsage() {
	fmt.Println(`usage: mailgraph [options]

  -h, --help           display this help and exit
  -v, --verbose        be verbose about what you do
  -V, --version        output version information and exit
  -c, --cat            causes the logfile to be only read and not monitored
  -l, --logfile f      monitor logfile f instead of /var/log/syslog
  -t, --logtype t      set logfile's type (default: syslog)
  -y, --year           starting year of the log file (default: current year)
      --host=HOST      use only entries for HOST (regexp) in syslog
  -d, --daemon         start collector in the background
  --daemon-pid=FILE    write PID to FILE instead of /var/run/mailgraph.pid
  --daemon-rrd=DIR     write RRDs to DIR instead of /var/www/mailgraph/rrd
  --daemon-log=FILE    write verbose-log to FILE instead of /var/log/mailgraph.log
  --ignore-localhost   ignore mail to/from localhost (used for virus scanner)
  --ignore-host=HOST   ignore mail to/from HOST regexp (used for virus scanner)
  --only-mail-rrd      update only the mail rrd
  --only-virus-rrd     update only the virus rrd
  --rrd-name=NAME      use NAME.rrd and NAME_virus.rrd for the rrd files
  --rbl-is-spam        count rbl rejects as spam
  --virbl-is-virus     count virbl rejects as viruses
      --serve          start HTTP server (default: true)
      --listen=ADDR    HTTP listen address (default: :8080)
      --hostname=HOST  hostname shown in graph title`)
}