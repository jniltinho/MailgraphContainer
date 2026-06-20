package collector

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidullrich/mailgraph/internal/config"
)

func TestProcessSampleLog(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	cfg.LogType = "syslog"
	cfg.Year = 2026
	cfg.Cat = true
	cfg.RRDDir = dir
	cfg.RRDName = "mailgraph"

	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	log := "Jun 20 10:00:01 mailhost postfix/smtpd[1001]: ABCD1234: client=unknown[192.168.1.10]\n" +
		"Jun 20 10:00:02 mailhost postfix/smtp[1002]: ABCD1234: to=<user@example.com>, relay=mail.example.com[10.0.0.1]:25, status=sent\n"
	if err := c.processReader(strings.NewReader(log)); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("rrdtool"); err != nil {
		t.Skip("rrdtool not installed")
	}
	if !c.inited {
		t.Fatal("expected collector to be initialized")
	}
	if _, err := os.Stat(filepath.Join(dir, "mailgraph.rrd")); err != nil {
		t.Fatalf("expected rrd file: %v", err)
	}
}