package collector

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/davidullrich/mailgraph/internal/config"
	"github.com/davidullrich/mailgraph/internal/rrd"
	"github.com/davidullrich/mailgraph/internal/syslog"
	"github.com/nxadm/tail"
)

type Collector struct {
	cfg         config.Config
	store       *rrd.Store
	currentMin  int64
	sum         rrd.Counters
	inited      bool
	ignoreHostRE []*regexp.Regexp
	hostFilter  *regexp.Regexp
}

func New(cfg config.Config) (*Collector, error) {
	c := &Collector{cfg: cfg}
	c.store = rrd.NewStore(cfg.RRDDir, cfg.RRDName, cfg.OnlyMailRRD, cfg.OnlyVirusRRD, cfg.Verbose)

	for _, ih := range cfg.IgnoreHosts {
		re, err := regexp.Compile(`(?i)\brelay=[^\s,]*` + ih)
		if err != nil {
			return nil, fmt.Errorf("invalid ignore-host %q: %w", ih, err)
		}
		c.ignoreHostRE = append(c.ignoreHostRE, re)
	}

	if cfg.HostFilter != "" {
		re, err := regexp.Compile(`(?i)^` + cfg.HostFilter + `$`)
		if err != nil {
			return nil, fmt.Errorf("invalid host filter: %w", err)
		}
		c.hostFilter = re
	}

	return c, nil
}

func (c *Collector) Run() error {
	if err := syslog.ValidateLogType(c.cfg.LogType); err != nil {
		return err
	}

	if c.cfg.Cat {
		f, err := os.Open(c.cfg.LogFile)
		if err != nil {
			return err
		}
		defer f.Close()
		return c.processReader(f)
	}

	if c.needsBootstrap() {
		f, err := os.Open(c.cfg.LogFile)
		if err == nil {
			_ = c.processReader(f)
			f.Close()
		}
	}

	t, err := tail.TailFile(c.cfg.LogFile, tail.Config{
		Follow:    true,
		ReOpen:    true,
		MustExist: false,
		Location:  &tail.SeekInfo{Offset: 0, Whence: io.SeekEnd},
	})
	if err != nil {
		return err
	}
	defer t.Cleanup()

	parser := syslog.NewParser(nil, c.cfg.LogType, c.cfg.Year)
	for line := range t.Lines {
		if line.Err != nil {
			return line.Err
		}
		entry, err := parser.ParseLine(line.Text)
		if err != nil {
			continue
		}
		if c.hostFilter != nil && !c.hostFilter.MatchString(entry.Host) {
			continue
		}
		c.processEntry(entry)
	}
	return nil
}

func (c *Collector) processReader(r io.Reader) error {
	parser := syslog.NewParser(r, c.cfg.LogType, c.cfg.Year)
	for {
		entry, err := parser.Next()
		if err == io.EOF {
			return c.flush()
		}
		if err != nil {
			return err
		}
		if c.hostFilter != nil && !c.hostFilter.MatchString(entry.Host) {
			continue
		}
		c.processEntry(entry)
	}
}

func (c *Collector) flush() error {
	if !c.inited || c.sum.IsZero() {
		return nil
	}
	return c.store.Update(c.currentMin, c.sum, false)
}

func (c *Collector) processEntry(e *syslog.Entry) { //nolint:gocyclo
	prog := e.Program
	text := e.Text
	ts := e.Timestamp.Unix()


	switch {
	case prog == "dovecot":
		if strings.Contains(text, "imap-login: Login") {
			c.event(ts, "dovecotloginsuccess")
		} else if strings.Contains(text, "unknown user") {
			c.event(ts, "dovecotloginfailed")
		}

	case prog == "policyd-spf":
		switch {
		case strings.Contains(text, "Received-SPF: None"), strings.Contains(text, "Received-SPF: Neutral"):
			c.event(ts, "spfnone")
		case strings.Contains(text, "Received-SPF: Pass"):
			c.event(ts, "spfpass")
		case strings.Contains(text, "Received-SPF:"):
			c.event(ts, "spffail")
		}

	case strings.HasPrefix(prog, "postfix/"):
		c.processPostfix(strings.TrimPrefix(prog, "postfix/"), text, ts)

	case prog == "sendmail" || prog == "sm-mta":
		c.processSendmail(text, ts)

	case prog == "exim":
		c.processExim(text, ts)

	case prog == "amavis" || prog == "amavisd":
		c.processAmavis(text, ts)

	case prog == "vagatefwd":
		if strings.HasPrefix(text, "VIRUS") {
			c.event(ts, "virus")
		}

	case prog == "hook":
		if strings.Contains(text, "* Virus") {
			c.event(ts, "virus")
		} else if strings.Contains(text, "contains spam") {
			c.event(ts, "spam")
		}

	case prog == "avgatefwd" || prog == "avmailgate.bin":
		if strings.HasPrefix(text, "Alert!") || strings.HasSuffix(text, "blocked.") {
			c.event(ts, "virus")
		}

	case prog == "avcheck":
		if strings.HasPrefix(text, "infected") {
			c.event(ts, "virus")
		}

	case prog == "spamd":
		if strings.Contains(text, "identified spam") {
			c.event(ts, "spam")
		} else if strings.Contains(text, "CLAMAV") {
			c.event(ts, "virus")
		}

	case prog == "dspam":
		if strings.Contains(text, "spam detected from") {
			c.event(ts, "spam")
		} else if strings.Contains(text, "infected message from") {
			c.event(ts, "virus")
		}

	case prog == "spamproxyd" || prog == "spampd":
		if strings.HasPrefix(strings.TrimSpace(text), "SPAM") || strings.Contains(text, "identified spam") {
			c.event(ts, "spam")
		}

	case prog == "drweb-postfix":
		if strings.Contains(text, "infected") {
			c.event(ts, "virus")
		}

	case prog == "BlackHole":
		if strings.Contains(text, "Virus") {
			c.event(ts, "virus")
		}
		if strings.Contains(text, "RBL") || strings.Contains(text, "Razor") || strings.Contains(text, "Spam") {
			c.event(ts, "spam")
		}

	case prog == "MailScanner":
		if strings.Contains(text, "Virus Scanning: Found") {
			c.event(ts, "virus")
		} else if strings.Contains(text, "Bounce to") {
			c.event(ts, "bounced")
		} else if m := regexp.MustCompile(`^Spam Checks: Found (\d+) spam messages`).FindStringSubmatch(text); len(m) == 2 {
			for i := 0; i < parseInt(m[1]); i++ {
				c.event(ts, "spam")
			}
		}

	case prog == "clamsmtpd":
		if strings.Contains(text, "status=VIRUS") {
			c.event(ts, "virus")
		}

	case prog == "clamav-milter":
		if strings.Contains(text, "Intercepted") || strings.Contains(text, "infected") || strings.Contains(text, "infected by") {
			c.event(ts, "virus")
		}

	case prog == "smtp-vilter":
		if strings.Contains(text, "clamd: found") {
			c.event(ts, "virus")
		}

	case prog == "opendmarc":
		switch {
		case strings.Contains(text, "pass"):
			c.event(ts, "dmarcpass")
		case strings.Contains(text, "none"):
			c.event(ts, "dmarcnone")
		case strings.Contains(text, "fail"):
			c.event(ts, "dmarcfail")
		}

	case prog == "opendkim":
		switch {
		case strings.Contains(text, "DKIM verification successful"):
			c.event(ts, "dkimpass")
		case strings.Contains(text, "no signature data"):
			c.event(ts, "dkimnone")
		case strings.Contains(text, "bad signature data"):
			c.event(ts, "dkimfail")
		}

	case prog == "avmilter":
		if strings.HasPrefix(text, "Alert!") || strings.HasSuffix(text, "blocked.") {
			c.event(ts, "virus")
		}

	case prog == "bogofilter":
		if strings.Contains(text, "Spam") {
			c.event(ts, "spam")
		}

	case prog == "filter-module":
		if strings.Contains(text, "spam_status=yes") || strings.Contains(text, "spam_status=spam") {
			c.event(ts, "spam")
		}

	case prog == "sta_scanner":
		if matched, _ := regexp.MatchString(`^[0-9A-F]+: virus`, text); matched {
			c.event(ts, "virus")
		}
	}
}

func (c *Collector) processPostfix(component, text string, ts int64) {
	switch component {
	case "smtp":
		if strings.Contains(text, "status=sent") {
			if c.cfg.IgnoreLocalhost && strings.Contains(text, "relay=") && strings.Contains(text, "[127.0.0.1]") {
				return
			}
			for _, re := range c.ignoreHostRE {
				if re.MatchString(text) {
					return
				}
			}
			c.event(ts, "sent")
		} else if strings.Contains(text, "status=bounced") {
			c.event(ts, "bounced")
		}

	case "local", "error":
		if strings.Contains(text, "status=bounced") {
			c.event(ts, "bounced")
		}

	case "smtpd":
		if m := regexp.MustCompile(`^[0-9A-Z]+: client=(\S+)`).FindStringSubmatch(text); len(m) == 2 {
			client := m[1]
			if c.cfg.IgnoreLocalhost && strings.HasSuffix(client, "[127.0.0.1]") {
				return
			}
			for _, ih := range c.cfg.IgnoreHosts {
				if matched, _ := regexp.MatchString(ih, client); matched {
					return
				}
			}
			c.event(ts, "received")
		} else if c.cfg.VirblIsVirus && regexp.MustCompile(`^(?:[0-9A-Z]+: |NOQUEUE: )?reject: .*: 554.* blocked using virbl\.dnsbl\.bit\.nl`).MatchString(text) {
			c.event(ts, "virus")
		} else if c.cfg.RBLIsSpam && regexp.MustCompile(`^(?:[0-9A-Z]+: |NOQUEUE: )?reject: .*: 554.* blocked using`).MatchString(text) {
			c.event(ts, "spam")
		} else if regexp.MustCompile(`^(?:[0-9A-Z]+: |NOQUEUE: )?reject: `).MatchString(text) {
			c.event(ts, "rejected")
		} else if regexp.MustCompile(`^(?:[0-9A-Z]+: |NOQUEUE: )?milter-reject: `).MatchString(text) {
			if strings.Contains(text, "Blocked by SpamAssassin") {
				c.event(ts, "spam")
			} else {
				c.event(ts, "rejected")
			}
		}

	case "cleanup":
		if regexp.MustCompile(`^[0-9A-Z]+: (?:reject|discard): `).MatchString(text) {
			c.event(ts, "rejected")
		}
	}
}

func (c *Collector) processSendmail(text string, ts int64) {
	switch {
	case strings.Contains(text, "mailer=local"), strings.Contains(text, "mailer=relay"):
		c.event(ts, "received")
	case strings.Contains(text, "stat=Sent"), strings.Contains(text, "mailer=esmtp"):
		c.event(ts, "sent")
	case strings.Contains(text, "stat=virus"):
		c.event(ts, "virus")
	case strings.Contains(text, "ruleset=check_relay"):
		if c.cfg.VirblIsVirus && strings.Contains(text, "ivirbl") {
			c.event(ts, "virus")
		} else if c.cfg.RBLIsSpam {
			c.event(ts, "spam")
		} else {
			c.event(ts, "rejected")
		}
	case strings.Contains(text, "ruleset=check_XS4ALL"),
		strings.Contains(text, "lost input channel"),
		strings.Contains(text, "ruleset=check_rcpt"),
		strings.Contains(text, "sender blocked"),
		strings.Contains(text, "sender denied"),
		strings.Contains(text, "recipient denied"),
		strings.Contains(text, "recipient unknown"),
		strings.Contains(text, "Milter:") && strings.Contains(text, "reject=55"):
		c.event(ts, "rejected")
	case strings.HasSuffix(strings.ToLower(text), "user unknown"):
		c.event(ts, "bounced")
	}
}

func (c *Collector) processExim(text string, ts int64) {
	switch {
	case regexp.MustCompile(`^[0-9a-zA-Z]{6}-[0-9a-zA-Z]{6}-[0-9a-zA-Z]{2} <= \S+`).MatchString(text):
		c.event(ts, "received")
	case regexp.MustCompile(`^[0-9a-zA-Z]{6}-[0-9a-zA-Z]{6}-[0-9a-zA-Z]{2} => \S+`).MatchString(text):
		c.event(ts, "sent")
	case strings.Contains(text, " rejected because ") && strings.Contains(text, " is in a black list at "):
		if c.cfg.RBLIsSpam {
			c.event(ts, "spam")
		} else {
			c.event(ts, "rejected")
		}
	case regexp.MustCompile(` rejected RCPT \S+: (Sender verify failed|Unknown user)`).MatchString(text):
		c.event(ts, "rejected")
	}
}

func (c *Collector) processAmavis(text string, ts int64) {
	switch {
	case regexp.MustCompile(`^\([\w-]+\) (Passed|Blocked) SPAM(?:MY)?\b`).MatchString(text):
		if !strings.Contains(text, "tag2=") {
			c.event(ts, "spam")
		}
	case regexp.MustCompile(`^\([\w-]+\) (Passed|Not-Delivered)\b.*\bquarantine spam`).MatchString(text):
		c.event(ts, "spam")
	case regexp.MustCompile(`^\([\w-]+\) (Passed |Blocked )?INFECTED\b`).MatchString(text):
		if !strings.Contains(text, "tag2=") {
			c.event(ts, "virus")
		}
	case regexp.MustCompile(`^\([\w-]+\) (Passed |Blocked )?BANNED\b`).MatchString(text):
		if !strings.Contains(text, "tag2=") {
			c.event(ts, "virus")
		}
	case strings.HasPrefix(text, "Virus found"):
		c.event(ts, "virus")
	}
}

func (c *Collector) event(ts int64, typ string) {
	if c.advance(ts) {
		c.increment(typ)
	}
}

func (c *Collector) advance(ts int64) bool {
	m := ts - ts%rrd.Step
	if !c.inited {
		current, err := c.store.Init(m)
		if err != nil {
			if c.cfg.Verbose {
				fmt.Fprintf(os.Stderr, "rrd init error: %v\n", err)
			}
			return false
		}
		c.currentMin = current
		c.inited = true
	}

	if m == c.currentMin {
		return true
	}
	if m < c.currentMin {
		return false
	}

	if err := c.store.Update(c.currentMin, c.sum, false); err != nil && c.cfg.Verbose {
		fmt.Fprintf(os.Stderr, "rrd update error: %v\n", err)
	}

	if m > c.currentMin+rrd.Step {
		if err := c.store.UpdateGap(c.currentMin+rrd.Step, m); err != nil && c.cfg.Verbose {
			fmt.Fprintf(os.Stderr, "rrd gap error: %v\n", err)
		}
	}

	c.currentMin = m
	c.sum = rrd.Counters{}
	return true
}

func (c *Collector) increment(typ string) {
	switch typ {
	case "sent":
		c.sum.Sent++
	case "received":
		c.sum.Received++
	case "bounced":
		c.sum.Bounced++
	case "rejected":
		c.sum.Rejected++
	case "spfnone":
		c.sum.SPFNone++
	case "spffail":
		c.sum.SPFFail++
	case "spfpass":
		c.sum.SPFPass++
	case "dmarcnone":
		c.sum.DMARCNone++
	case "dmarcfail":
		c.sum.DMARCFail++
	case "dmarcpass":
		c.sum.DMARCPass++
	case "dkimnone":
		c.sum.DKIMNone++
	case "dkimfail":
		c.sum.DKIMFail++
	case "dkimpass":
		c.sum.DKIMPass++
	case "virus":
		c.sum.Virus++
	case "spam":
		c.sum.Spam++
	case "dovecotloginsuccess":
		c.sum.DovecotLoginSuccess++
	case "dovecotloginfailed":
		c.sum.DovecotLoginFailed++
	}
}

func (c *Collector) needsBootstrap() bool {
	if _, err := os.Stat(c.store.MailPath()); err == nil {
		return false
	}
	if _, err := os.Stat(c.store.VirusPath()); err == nil {
		return false
	}
	return true
}

func parseInt(s string) int {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			break
		}
		n = n*10 + int(ch-'0')
	}
	return n
}