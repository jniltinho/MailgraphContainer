// Package charts renders interactive go-echarts graphs from RRD data.
package charts

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"

	"mailgraph/internal/rrd"
)

// Period describes a predefined graph time range.
type Period struct {
	Title        string
	Slug         string
	Seconds      int
	FromMidnight bool
}

// Periods lists the predefined time ranges available in the web UI.
var Periods = []Period{
	{Title: "Today", Slug: "today", FromMidnight: true},
	{Title: "Last Day", Slug: "last-day", Seconds: 3600 * 24},
	{Title: "Last Week", Slug: "last-week", Seconds: 3600 * 24 * 7},
	{Title: "Last 2 Weeks", Slug: "last-2-weeks", Seconds: 3600 * 24 * 7 * 2},
	{Title: "Last Month", Slug: "last-month", Seconds: 3600 * 24 * 31},
	{Title: "Last 2 Month", Slug: "last-2-month", Seconds: 3600 * 24 * 31 * 2},
	{Title: "Last Year", Slug: "last-year", Seconds: 3600 * 24 * 365},
	{Title: "Last 2 Years", Slug: "last-2-years", Seconds: 3600 * 24 * 365 * 2},
}

// PeriodIndex returns the period index for slug, or false when unknown.
func PeriodIndex(slug string) (int, bool) {
	for i, p := range Periods {
		if p.Slug == slug {
			return i, true
		}
	}
	return 0, false
}

const (
	// TypeTraffic is the sent/received traffic chart.
	TypeTraffic = "n"
	// TypeErrors is the bounced, rejected, virus, and spam chart.
	TypeErrors = "e"
	// TypeSPF is the SPF result chart.
	TypeSPF = "s"
	// TypeDMARC is the DMARC result chart.
	TypeDMARC = "d"
	// TypeDKIM is the DKIM result chart.
	TypeDKIM = "k"
	// TypeDovecot is the Dovecot login chart.
	TypeDovecot = "v"
)

var colors = map[string]string{
	"sent":                "#000099",
	"recv":                "#009900",
	"spfnone":             "#000AAA",
	"spffail":             "#12FF0A",
	"spfpass":             "#D15400",
	"dmarcnone":           "#FFFF00",
	"dmarcfail":           "#FF00EA",
	"dmarcpass":           "#00FFD5",
	"dkimnone":            "#3013EC",
	"dkimfail":            "#006B3A",
	"dkimpass":            "#491503",
	"rejected":            "#AA0000",
	"bounced":             "#000000",
	"virus":               "#DDBB00",
	"spam":                "#999999",
	"dovecotloginsuccess": "#999999",
	"dovecotloginfailed":  "#006400",
}

// Generator builds chart HTML fragments from an RRD store.
type Generator struct {
	store *rrd.Store
}

// NewGenerator creates a chart Generator backed by store.
func NewGenerator(store *rrd.Store) *Generator {
	return &Generator{store: store}
}

// Render returns the HTML for the chart identified by period index and chartType.
func (g *Generator) Render(period int, chartType string) (string, error) {
	if period < 0 || period >= len(Periods) {
		return "", fmt.Errorf("invalid period %d", period)
	}

	p := Periods[period]

	switch chartType {
	case TypeTraffic:
		return g.renderTraffic(p)
	case TypeErrors:
		return g.renderErrors(p)
	case TypeSPF:
		return g.renderSPF(p)
	case TypeDMARC:
		return g.renderDMARC(p)
	case TypeDKIM:
		return g.renderDKIM(p)
	case TypeDovecot:
		return g.renderDovecot(p)
	default:
		return "", fmt.Errorf("invalid chart type %q", chartType)
	}
}

func (g *Generator) fetch(path string, period Period) ([]rrd.DataPoint, error) {
	if period.FromMidnight {
		return g.store.FetchToday(path)
	}
	return g.store.Fetch(path, period.Seconds)
}

func (g *Generator) renderTraffic(period Period) (string, error) {
	points, err := g.fetch(g.store.MailPath(), period)
	if err != nil {
		if isMissingRRD(err) {
			return emptyChart(period.Title, "msgs/min", "No data yet"), nil
		}
		return "", err
	}

	labels, series := buildSeries(points, period, "sent", "recv")
	line := baseLine(period.Title, "msgs/min", labels)
	line.AddSeries("Sent", series[0],
		charts.WithAreaStyleOpts(opts.AreaStyle{Color: colors["sent"]}),
		charts.WithLineStyleOpts(opts.LineStyle{Color: colors["sent"]}),
	)
	line.AddSeries("Received", series[1],
		charts.WithLineStyleOpts(opts.LineStyle{Color: colors["recv"], Width: 2}),
	)
	return renderChart(line)
}

func (g *Generator) renderErrors(period Period) (string, error) {
	mailPts, mailErr := g.fetch(g.store.MailPath(), period)
	virusPts, virusErr := g.fetch(g.store.VirusPath(), period)
	if isMissingRRD(mailErr) && isMissingRRD(virusErr) {
		return emptyChart(period.Title+" - Errors", "msgs/min", "No data yet"), nil
	}
	if mailErr != nil && !isMissingRRD(mailErr) {
		return "", mailErr
	}
	if virusErr != nil && !isMissingRRD(virusErr) {
		return "", virusErr
	}

	labels := labelsFrom(mailPts, period)
	if len(labels) == 0 {
		labels = labelsFrom(virusPts, period)
	}

	line := baseLine(period.Title+" - Errors", "msgs/min", labels)
	line.AddSeries("Bounced", valuesFor(mailPts, "bounced"),
		charts.WithAreaStyleOpts(opts.AreaStyle{Color: colors["bounced"]}),
		charts.WithLineStyleOpts(opts.LineStyle{Color: colors["bounced"]}),
	)
	line.AddSeries("Viruses", valuesFor(virusPts, "virus"),
		charts.WithAreaStyleOpts(opts.AreaStyle{Color: colors["virus"]}),
		charts.WithLineStyleOpts(opts.LineStyle{Color: colors["virus"]}),
	)
	line.AddSeries("Spam", valuesFor(virusPts, "spam"),
		charts.WithAreaStyleOpts(opts.AreaStyle{Color: colors["spam"]}),
		charts.WithLineStyleOpts(opts.LineStyle{Color: colors["spam"]}),
	)
	line.AddSeries("Rejected", valuesFor(mailPts, "rejected"),
		charts.WithLineStyleOpts(opts.LineStyle{Color: colors["rejected"], Width: 2}),
	)
	return renderChart(line)
}

func (g *Generator) renderSPF(period Period) (string, error) {
	return g.renderTriple(g.store.MailPath(), period, period.Title+" - SPF", "msgs/min", "spfpass", "spfnone", "spffail", "SPF pass", "SPF none", "SPF fail")
}

func (g *Generator) renderDMARC(period Period) (string, error) {
	return g.renderTriple(g.store.MailPath(), period, period.Title+" - DMARC", "msgs/min", "dmarcpass", "dmarcnone", "dmarcfail", "DMARC pass", "DMARC none", "DMARC fail")
}

func (g *Generator) renderDKIM(period Period) (string, error) {
	return g.renderTriple(g.store.MailPath(), period, period.Title+" - DKIM", "msgs/min", "dkimpass", "dkimnone", "dkimfail", "DKIM pass", "DKIM none", "DKIM fail")
}

func (g *Generator) renderDovecot(period Period) (string, error) {
	points, err := g.fetch(g.store.DovecotPath(), period)
	if err != nil {
		if isMissingRRD(err) {
			return emptyChart(period.Title+" - Dovecot", "logins/min", "No data yet"), nil
		}
		return "", err
	}

	labels, series := buildSeries(points, period, "dovecotloginsuccess", "dovecotloginfailed")
	line := baseLine(period.Title+" - Dovecot", "logins/min", labels)
	line.AddSeries("Dovecot logins successful", series[0],
		charts.WithAreaStyleOpts(opts.AreaStyle{Color: colors["dovecotloginsuccess"]}),
		charts.WithLineStyleOpts(opts.LineStyle{Color: colors["dovecotloginsuccess"]}),
	)
	line.AddSeries("Dovecot logins failed", series[1],
		charts.WithLineStyleOpts(opts.LineStyle{Color: colors["dovecotloginfailed"], Width: 2}),
	)
	return renderChart(line)
}

func (g *Generator) renderTriple(path string, period Period, title, yLabel, k1, k2, k3, n1, n2, n3 string) (string, error) {
	points, err := g.fetch(path, period)
	if err != nil {
		if isMissingRRD(err) {
			return emptyChart(title, yLabel, "No data yet"), nil
		}
		return "", err
	}

	labels, series := buildSeries(points, period, k1, k2, k3)
	line := baseLine(title, yLabel, labels)
	line.AddSeries(n1, series[0],
		charts.WithAreaStyleOpts(opts.AreaStyle{Color: colors[k1]}),
		charts.WithLineStyleOpts(opts.LineStyle{Color: colors[k1]}),
	)
	line.AddSeries(n2, series[1],
		charts.WithAreaStyleOpts(opts.AreaStyle{Color: colors[k2]}),
		charts.WithLineStyleOpts(opts.LineStyle{Color: colors[k2]}),
	)
	line.AddSeries(n3, series[2],
		charts.WithLineStyleOpts(opts.LineStyle{Color: colors[k3], Width: 2}),
	)
	return renderChart(line)
}

func baseLine(title, yLabel string, labels []string) *charts.Line {
	line := charts.NewLine()
	line.SetXAxis(labels)
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Width: "900px", Height: "200px"}),
		charts.WithTitleOpts(opts.Title{Title: title, Left: "center"}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true), Top: "bottom"}),
		charts.WithGridOpts(opts.Grid{Left: "3%", Right: "4%", Bottom: "15%", ContainLabel: opts.Bool(true)}),
		charts.WithXAxisOpts(opts.XAxis{
			Type: "category",
			AxisLabel: &opts.AxisLabel{
				Rotate:      45,
				HideOverlap: opts.Bool(true),
			},
		}),
		charts.WithYAxisOpts(opts.YAxis{Name: yLabel, Type: "value", Min: opts.Float(0)}),
	)
	return line
}

func formatAxisLabel(period Period, t time.Time) string {
	switch period.Slug {
	case "today", "last-day":
		return t.Format("15:04")
	case "last-week", "last-2-weeks":
		// Match legacy mailgraph.cgi / rrdtool graph labels (e.g. "Mon 15 Jun").
		return t.Format("Mon 2 Jan")
	case "last-year", "last-2-years":
		return t.Format("2006-01-02")
	default:
		return t.Format("01-02")
	}
}

func axisLabelsFor(points []rrd.DataPoint, period Period) []string {
	labels := make([]string, len(points))
	dedupeDay := period.Slug == "last-week" || period.Slug == "last-2-weeks"
	var lastDay string
	for i, p := range points {
		day := p.Timestamp.Format("2006-01-02")
		if dedupeDay && day == lastDay {
			labels[i] = ""
			continue
		}
		lastDay = day
		labels[i] = formatAxisLabel(period, p.Timestamp)
	}
	return labels
}

func buildSeries(points []rrd.DataPoint, period Period, keys ...string) ([]string, [][]opts.LineData) {
	labels := axisLabelsFor(points, period)
	series := make([][]opts.LineData, len(keys))
	for i := range keys {
		series[i] = make([]opts.LineData, len(points))
	}
	for i, p := range points {
		for j, key := range keys {
			series[j][i] = opts.LineData{Value: rrd.RatePerMinute(p.Values[key])}
		}
	}
	return labels, series
}

func labelsFrom(points []rrd.DataPoint, period Period) []string {
	return axisLabelsFor(points, period)
}

func valuesFor(points []rrd.DataPoint, key string) []opts.LineData {
	data := make([]opts.LineData, len(points))
	for i, p := range points {
		data[i] = opts.LineData{Value: rrd.RatePerMinute(p.Values[key])}
	}
	return data
}

const chartFrameCSS = `<style>html,body{margin:0;padding:0;overflow:hidden;background:#fff}.container{margin:0!important;padding:0}</style>`

func wrapChartHTML(html string) string {
	if idx := strings.Index(html, "</head>"); idx >= 0 {
		return html[:idx] + chartFrameCSS + html[idx:]
	}
	return chartFrameCSS + html
}

func renderChart(line *charts.Line) (string, error) {
	var buf bytes.Buffer
	if err := line.Render(io.MultiWriter(&buf)); err != nil {
		return "", err
	}
	return wrapChartHTML(buf.String()), nil
}

func emptyChart(title, yLabel, message string) string {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Width: "900px", Height: "200px"}),
		charts.WithTitleOpts(opts.Title{Title: title, Subtitle: message, Left: "center"}),
		charts.WithYAxisOpts(opts.YAxis{Name: yLabel, Min: opts.Float(0)}),
	)
	var buf bytes.Buffer
	_ = line.Render(&buf)
	return wrapChartHTML(buf.String())
}

func isMissingRRD(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "No such file") || strings.Contains(err.Error(), "not found")
}