package syslog

import (
	"io"
	"strings"
	"testing"
)

func TestParseLinePostfix(t *testing.T) {
	p := NewParser(nil, "syslog", 2026)
	entry, err := p.ParseLine("Jun 20 10:00:01 mailhost postfix/smtpd[1001]: ABCD1234: client=unknown[192.168.1.10]")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if entry.Program != "postfix/smtpd" {
		t.Fatalf("program = %q", entry.Program)
	}
	if entry.Text == "" {
		t.Fatal("expected non-empty message text")
	}
}

func TestParseSMTPLine(t *testing.T) {
	p := NewParser(nil, "syslog", 2026)
	entry, err := p.ParseLine("Jun 20 10:00:02 mailhost postfix/smtp[1002]: ABCD1234: to=<user@example.com>, status=sent")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if entry.Program != "postfix/smtp" {
		t.Fatalf("program = %q", entry.Program)
	}
}

func TestNextReadsMultipleLines(t *testing.T) {
	log := "Jun 20 10:00:01 mailhost postfix/smtpd[1001]: ABCD1234: client=unknown[192.168.1.10]\n" +
		"Jun 20 10:00:02 mailhost postfix/smtp[1002]: ABCD1234: to=<user@example.com>, status=sent\n"
	p := NewParser(strings.NewReader(log), "syslog", 2026)

	e1, err := p.Next()
	if err != nil {
		t.Fatalf("first line: %v", err)
	}
	if e1.Program != "postfix/smtpd" {
		t.Fatalf("program1 = %q", e1.Program)
	}

	e2, err := p.Next()
	if err != nil {
		t.Fatalf("second line: %v", err)
	}
	if e2.Program != "postfix/smtp" {
		t.Fatalf("program2 = %q", e2.Program)
	}

	_, err = p.Next()
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}