package syslog

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var months = map[string]time.Month{
	"Jan": time.January, "Feb": time.February, "Mar": time.March,
	"Apr": time.April, "May": time.May, "Jun": time.June,
	"Jul": time.July, "Aug": time.August, "Sep": time.September,
	"Oct": time.October, "Nov": time.November, "Dec": time.December,
	"jan": time.January, "feb": time.February, "mar": time.March,
	"apr": time.April, "may": time.May, "jun": time.June,
	"jul": time.July, "aug": time.August, "sep": time.September,
	"oct": time.October, "nov": time.November, "dec": time.December,
}

var (
	syslogLine = regexp.MustCompile(`^(\S{3})\s+(\d+)\s+(\d+):(\d+):(\d+)(?:\s+<\w+\.\w+>)?\s+([-\w\.\@:]+)\s+(?:\[LOG_[A-Z]+\]\s+)?(.*)$`)
	programLine = regexp.MustCompile(`^([^:]+?)(?:\[(\d+)\])?:\s+(?:\[ID\ (\d+)\ ([a-z0-9]+)\.([a-z]+)\]\ )?(.*)$`)
	metalogLine = regexp.MustCompile(`^(\S{3})\s+(\d+)\s+(\d+):(\d+):(\d+)\s+(.*)$`)
	metalogText = regexp.MustCompile(`^\[(.*?)\]\s+(.*)$`)
)

type Entry struct {
	Timestamp time.Time
	Host      string
	Program   string
	PID       string
	Text      string
}

type Parser struct {
	reader    io.Reader
	scanner   *bufio.Scanner
	logType   string
	year      int
	lastMonth time.Month
	location  *time.Location
}

func NewParser(r io.Reader, logType string, year int) *Parser {
	if logType == "" {
		logType = "syslog"
	}
	if year == 0 {
		year = time.Now().Year()
	}
	p := &Parser{
		reader:   r,
		logType:  logType,
		year:     year,
		location: time.Local,
	}
	if r != nil {
		p.scanner = bufio.NewScanner(r)
	}
	return p
}

func (p *Parser) Next() (*Entry, error) {
	switch p.logType {
	case "metalog":
		return p.nextMetalog()
	default:
		return p.nextSyslog()
	}
}

func (p *Parser) ParseLine(line string) (*Entry, error) {
	switch p.logType {
	case "metalog":
		return p.parseMetalogLine(line)
	default:
		return p.parseSyslogLine(line)
	}
}

func (p *Parser) nextSyslog() (*Entry, error) {
	if p.scanner == nil {
		return nil, io.EOF
	}
	for p.scanner.Scan() {
		entry, err := p.parseSyslogLine(p.scanner.Text())
		if err != nil {
			continue
		}
		if entry != nil {
			return entry, nil
		}
	}
	if err := p.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (p *Parser) parseSyslogLine(line string) (*Entry, error) {
	m := syslogLine.FindStringSubmatch(line)
	if m == nil {
		return nil, fmt.Errorf("not syslog")
	}

	mon, ok := months[m[1]]
	if !ok {
		return nil, fmt.Errorf("unknown month")
	}
	day, _ := strconv.Atoi(m[2])
	hour, _ := strconv.Atoi(m[3])
	min, _ := strconv.Atoi(m[4])
	sec, _ := strconv.Atoi(m[5])
	host := m[6]
	text := m[7]

	p.adjustYear(mon)
	ts := time.Date(p.year, mon, day, hour, min, sec, 0, p.location)

	if text == "-- MARK --" {
		return nil, fmt.Errorf("skip")
	}
	if strings.HasPrefix(text, "last message repeated") || strings.HasPrefix(text, "above message repeats") {
		return nil, fmt.Errorf("skip")
	}

	text = strings.TrimPrefix(text, host+" ")
	text = strings.TrimPrefix(text, ": ")

	pm := programLine.FindStringSubmatch(text)
	if pm == nil {
		return nil, fmt.Errorf("not program")
	}

	return &Entry{
		Timestamp: ts,
		Host:      host,
		Program:   pm[1],
		PID:       pm[2],
		Text:      pm[len(pm)-1],
	}, nil
}

func (p *Parser) nextMetalog() (*Entry, error) {
	if p.scanner == nil {
		return nil, io.EOF
	}
	for p.scanner.Scan() {
		entry, err := p.parseMetalogLine(p.scanner.Text())
		if err != nil {
			continue
		}
		if entry != nil {
			return entry, nil
		}
	}
	if err := p.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (p *Parser) parseMetalogLine(line string) (*Entry, error) {
	m := metalogLine.FindStringSubmatch(line)
	if m == nil {
		return nil, fmt.Errorf("not metalog")
	}

	mon, ok := months[m[1]]
	if !ok {
		return nil, fmt.Errorf("unknown month")
	}
	day, _ := strconv.Atoi(m[2])
	hour, _ := strconv.Atoi(m[3])
	min, _ := strconv.Atoi(m[4])
	sec, _ := strconv.Atoi(m[5])
	text := m[6]

	p.adjustYear(mon)
	ts := time.Date(p.year, mon, day, hour, min, sec, 0, p.location)

	tm := metalogText.FindStringSubmatch(text)
	if tm == nil {
		return nil, fmt.Errorf("not metalog text")
	}

	return &Entry{
		Timestamp: ts,
		Host:      "localhost",
		Program:   tm[1],
		Text:      tm[2],
	}, nil
}

func (p *Parser) adjustYear(mon time.Month) {
	if p.lastMonth == time.December && mon == time.January {
		p.year++
	} else if p.lastMonth != time.December && mon == time.December {
		// allow year decrement at year boundary when reading old logs
	}
	p.lastMonth = mon
}

func (p *Parser) SetReader(r io.Reader) {
	p.reader = r
	if r != nil {
		p.scanner = bufio.NewScanner(r)
	}
}

func ParseLine(logType string, year int, line string) (*Entry, error) {
	p := NewParser(strings.NewReader(line+"\n"), logType, year)
	return p.Next()
}

func FormatTimestamp(t time.Time) string {
	return t.Format("Mon Jan 2 15:04:05 2006")
}

func ValidateLogType(logType string) error {
	switch logType {
	case "syslog", "metalog":
		return nil
	default:
		return fmt.Errorf("unknown log type %q", logType)
	}
}