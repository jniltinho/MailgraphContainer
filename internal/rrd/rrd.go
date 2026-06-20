// Package rrd wraps rrdtool create, update, and fetch operations for mail statistics.
package rrd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// Step is the RRD step size in seconds (one-minute buckets).
	Step = 60
	// XPoints is the number of points per graph axis sample.
	XPoints = 540
	// PointsSample is the downsampling factor for graph rendering.
	PointsSample = 3
)

// Store manages mailgraph RRD files in a directory.
type Store struct {
	dir       string
	mailRRD   string
	virusRRD  string
	dovecotRRD string
	onlyMail  bool
	onlyVirus bool
	verbose   bool
}

// DataPoint is a single RRD fetch sample.
type DataPoint struct {
	Timestamp time.Time
	Values    map[string]float64
}

// NewStore creates a Store for RRD files named name in dir.
func NewStore(dir, name string, onlyMail, onlyVirus, verbose bool) *Store {
	return &Store{
		dir:        dir,
		mailRRD:    name + ".rrd",
		virusRRD:   name + "_virus.rrd",
		dovecotRRD: name + "_dovecot.rrd",
		onlyMail:   onlyMail,
		onlyVirus:  onlyVirus,
		verbose:    verbose,
	}
}

// MailPath returns the path to the main mail RRD file.
func (s *Store) MailPath() string { return filepath.Join(s.dir, s.mailRRD) }

// VirusPath returns the path to the virus and spam RRD file.
func (s *Store) VirusPath() string { return filepath.Join(s.dir, s.virusRRD) }

// DovecotPath returns the path to the Dovecot login RRD file.
func (s *Store) DovecotPath() string { return filepath.Join(s.dir, s.dovecotRRD) }

// Init creates missing RRD files and returns the next update timestamp.
func (s *Store) Init(startMinute int64) (int64, error) {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return 0, err
	}

	rows := XPoints / PointsSample
	realRows := int(float64(rows) * 1.1)
	daySteps := 3600 * 24 / (Step * rows)
	weekSteps := daySteps * 7
	monthSteps := weekSteps * 5
	yearSteps := monthSteps * 12

	current := startMinute

	if !s.onlyVirus {
		if _, err := os.Stat(s.MailPath()); os.IsNotExist(err) {
			args := []string{
				"create", s.MailPath(),
				"--start", strconv.FormatInt(startMinute, 10),
				"--step", strconv.Itoa(Step),
				"DS:sent:ABSOLUTE:" + strconv.Itoa(Step*2) + ":0:U",
				"DS:recv:ABSOLUTE:" + strconv.Itoa(Step*2) + ":0:U",
				"DS:bounced:ABSOLUTE:" + strconv.Itoa(Step*2) + ":0:U",
				"DS:rejected:ABSOLUTE:" + strconv.Itoa(Step*2) + ":0:U",
				"DS:spfnone:ABSOLUTE:" + strconv.Itoa(Step*2) + ":0:U",
				"DS:spffail:ABSOLUTE:" + strconv.Itoa(Step*2) + ":0:U",
				"DS:spfpass:ABSOLUTE:" + strconv.Itoa(Step*2) + ":0:U",
				"DS:dmarcnone:ABSOLUTE:" + strconv.Itoa(Step*2) + ":0:U",
				"DS:dmarcfail:ABSOLUTE:" + strconv.Itoa(Step*2) + ":0:U",
				"DS:dmarcpass:ABSOLUTE:" + strconv.Itoa(Step*2) + ":0:U",
				"DS:dkimnone:ABSOLUTE:" + strconv.Itoa(Step*2) + ":0:U",
				"DS:dkimfail:ABSOLUTE:" + strconv.Itoa(Step*2) + ":0:U",
				"DS:dkimpass:ABSOLUTE:" + strconv.Itoa(Step*2) + ":0:U",
				fmt.Sprintf("RRA:AVERAGE:0.5:%d:%d", daySteps, realRows),
				fmt.Sprintf("RRA:AVERAGE:0.5:%d:%d", weekSteps, realRows),
				fmt.Sprintf("RRA:AVERAGE:0.5:%d:%d", monthSteps, realRows),
				fmt.Sprintf("RRA:AVERAGE:0.5:%d:%d", yearSteps, realRows),
				fmt.Sprintf("RRA:MAX:0.5:%d:%d", daySteps, realRows),
				fmt.Sprintf("RRA:MAX:0.5:%d:%d", weekSteps, realRows),
				fmt.Sprintf("RRA:MAX:0.5:%d:%d", monthSteps, realRows),
				fmt.Sprintf("RRA:MAX:0.5:%d:%d", yearSteps, realRows),
			}
			if err := runRRD(args...); err != nil {
				return 0, err
			}
			current = startMinute
		} else {
			last, err := s.last(s.MailPath())
			if err != nil {
				return 0, err
			}
			current = last + Step
		}
	}

	if !s.onlyMail {
		if _, err := os.Stat(s.VirusPath()); os.IsNotExist(err) {
			args := []string{
				"create", s.VirusPath(),
				"--start", strconv.FormatInt(startMinute, 10),
				"--step", strconv.Itoa(Step),
				"DS:virus:ABSOLUTE:" + strconv.Itoa(Step*2) + ":0:U",
				"DS:spam:ABSOLUTE:" + strconv.Itoa(Step*2) + ":0:U",
				fmt.Sprintf("RRA:AVERAGE:0.5:%d:%d", daySteps, realRows),
				fmt.Sprintf("RRA:AVERAGE:0.5:%d:%d", weekSteps, realRows),
				fmt.Sprintf("RRA:AVERAGE:0.5:%d:%d", monthSteps, realRows),
				fmt.Sprintf("RRA:AVERAGE:0.5:%d:%d", yearSteps, realRows),
				fmt.Sprintf("RRA:MAX:0.5:%d:%d", daySteps, realRows),
				fmt.Sprintf("RRA:MAX:0.5:%d:%d", weekSteps, realRows),
				fmt.Sprintf("RRA:MAX:0.5:%d:%d", monthSteps, realRows),
				fmt.Sprintf("RRA:MAX:0.5:%d:%d", yearSteps, realRows),
			}
			if err := runRRD(args...); err != nil {
				return 0, err
			}
		}

		if _, err := os.Stat(s.DovecotPath()); os.IsNotExist(err) {
			args := []string{
				"create", s.DovecotPath(),
				"--start", strconv.FormatInt(startMinute, 10),
				"--step", strconv.Itoa(Step),
				"DS:dovecotloginsuccess:ABSOLUTE:" + strconv.Itoa(Step*2) + ":0:U",
				"DS:dovecotloginfailed:ABSOLUTE:" + strconv.Itoa(Step*2) + ":0:U",
				fmt.Sprintf("RRA:AVERAGE:0.5:%d:%d", daySteps, realRows),
				fmt.Sprintf("RRA:AVERAGE:0.5:%d:%d", weekSteps, realRows),
				fmt.Sprintf("RRA:AVERAGE:0.5:%d:%d", monthSteps, realRows),
				fmt.Sprintf("RRA:AVERAGE:0.5:%d:%d", yearSteps, realRows),
				fmt.Sprintf("RRA:MAX:0.5:%d:%d", daySteps, realRows),
				fmt.Sprintf("RRA:MAX:0.5:%d:%d", weekSteps, realRows),
				fmt.Sprintf("RRA:MAX:0.5:%d:%d", monthSteps, realRows),
				fmt.Sprintf("RRA:MAX:0.5:%d:%d", yearSteps, realRows),
			}
			if err := runRRD(args...); err != nil {
				return 0, err
			}
		}
	}

	return current, nil
}

// IsZero reports whether c contains no counted events.
func (c Counters) IsZero() bool {
	return c == Counters{}
}

// Counters holds per-minute mail event totals before flushing to RRD.
type Counters struct {
	Sent, Received, Bounced, Rejected                int
	SPFNone, SPFFail, SPFPass                        int
	DMARCNone, DMARCFail, DMARCPass                  int
	DKIMNone, DKIMFail, DKIMPass                     int
	Virus, Spam                                      int
	DovecotLoginSuccess, DovecotLoginFailed          int
}

// Update writes counter values for minute into the RRD files.
func (s *Store) Update(minute int64, c Counters, fillGaps bool) error {
	if !s.onlyVirus {
		value := fmt.Sprintf("%d:%d:%d:%d:%d:%d:%d:%d:%d:%d:%d:%d:%d:%d",
			minute, c.Sent, c.Received, c.Bounced, c.Rejected,
			c.SPFNone, c.SPFFail, c.SPFPass,
			c.DMARCNone, c.DMARCFail, c.DMARCPass,
			c.DKIMNone, c.DKIMFail, c.DKIMPass)
		if s.verbose {
			fmt.Println("update", value)
		}
		if err := runRRD("update", s.MailPath(), value); err != nil {
			return err
		}
	}

	if !s.onlyMail {
		value := fmt.Sprintf("%d:%d:%d", minute, c.Virus, c.Spam)
		if s.verbose {
			fmt.Println("update virus", value)
		}
		if err := runRRD("update", s.VirusPath(), value); err != nil {
			return err
		}

		value = fmt.Sprintf("%d:%d:%d", minute, c.DovecotLoginSuccess, c.DovecotLoginFailed)
		if s.verbose {
			fmt.Println("update dovecot", value)
		}
		if err := runRRD("update", s.DovecotPath(), value); err != nil {
			return err
		}
	}

	if fillGaps {
		_ = fillGaps // handled by caller via UpdateGap
	}

	return nil
}

// UpdateGap fills empty minute buckets between from and to with zero values.
func (s *Store) UpdateGap(from, to int64) error {
	for sm := from; sm < to; sm += Step {
		if !s.onlyVirus {
			value := fmt.Sprintf("%d:0:0:0:0:0:0:0:0:0:0:0:0", sm)
			if s.verbose {
				fmt.Println("update", value, "(SKIP)")
			}
			if err := runRRD("update", s.MailPath(), value); err != nil {
				return err
			}
		}
		if !s.onlyMail {
			if err := runRRD("update", s.VirusPath(), fmt.Sprintf("%d:0:0", sm)); err != nil {
				return err
			}
			if err := runRRD("update", s.DovecotPath(), fmt.Sprintf("%d:0:0", sm)); err != nil {
				return err
			}
		}
	}
	return nil
}

// Fetch returns averaged RRD samples from the last seconds interval.
func (s *Store) Fetch(path string, seconds int) ([]DataPoint, error) {
	return s.FetchRange(path, fmt.Sprintf("-%d", seconds))
}

// FetchToday returns averaged RRD samples from local midnight until now.
func (s *Store) FetchToday(path string) ([]DataPoint, error) {
	return s.FetchRange(path, "midnight")
}

// FetchRange returns averaged RRD samples from start until now.
// start accepts rrdtool time specs such as "-86400", "midnight", or a Unix timestamp.
func (s *Store) FetchRange(path string, start string) ([]DataPoint, error) {
	args := []string{
		"fetch", path, "AVERAGE",
		"--start", start,
		"--end", "now",
	}
	out, err := exec.Command("rrdtool", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("rrdtool fetch: %w: %s", err, string(out))
	}

	return parseFetch(string(out))
}

func (s *Store) last(path string) (int64, error) {
	out, err := exec.Command("rrdtool", "last", path).CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("rrdtool last: %w: %s", err, string(out))
	}
	return strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
}

func runRRD(args ...string) error {
	out, err := exec.Command("rrdtool", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("rrdtool %s: %w: %s", strings.Join(args, " "), err, string(out))
	}
	return nil
}

func parseFetch(output string) ([]DataPoint, error) {
	scanner := bufio.NewScanner(strings.NewReader(output))
	var headers []string
	var points []DataPoint

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "nan") && !strings.Contains(line, ":") {
			continue
		}
		if !strings.Contains(line, ":") {
			fields := strings.Fields(line)
			if len(fields) > 0 && fields[0] != "" {
				headers = fields
			}
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		ts, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		if err != nil {
			continue
		}

		values := strings.Fields(strings.TrimSpace(parts[1]))
		dp := DataPoint{
			Timestamp: time.Unix(ts, 0),
			Values:    make(map[string]float64, len(headers)),
		}
		for i, h := range headers {
			if i >= len(values) {
				break
			}
			v := strings.TrimSpace(values[i])
			if v == "nan" || v == "-nan" {
				dp.Values[h] = 0
				continue
			}
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				dp.Values[h] = 0
				continue
			}
			dp.Values[h] = f
		}
		points = append(points, dp)
	}

	return points, scanner.Err()
}

// RatePerMinute converts a per-step average to messages per minute.
func RatePerMinute(v float64) float64 {
	return v * 60
}