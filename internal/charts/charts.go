package charts

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"

	"github.com/davidullrich/mailgraph/internal/rrd"
)

var Periods = []struct {
	Title   string
	Seconds int
}{
	{Title: "Last Day", Seconds: 3600 * 24},
	{Title: "Last Week", Seconds: 3600 * 24 * 7},
	{Title: "Last 2 Weeks", Seconds: 3600 * 24 * 7 * 2},
	{Title: "Last Month", Seconds: 3600 * 24 * 31},
	{Title: "Last 2 Month", Seconds: 3600 * 24 * 31 * 2},
	{Title: "Last Year", Seconds: 3600 * 24 * 365},
	{Title: "Last 2 Years", Seconds: 3600 * 24 * 365 * 2},
}

const (
	TypeTraffic = "n"
	TypeErrors  = "e"
	TypeSPF     = "s"
	TypeDMARC   = "d"
	TypeDKIM    = "k"
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

type Generator struct {
	store *rrd.Store
}

func NewGenerator(store *rrd.Store) *Generator {
	return &Generator{store: store}
}

func (g *Generator) Render(period int, chartType string) (string, error) {
	if period < 0 || period >= len(Periods) {
		return "", fmt.Errorf("invalid period %d", period)
	}

	seconds := Periods[period].Seconds
	title := Periods[period].Title

	switch chartType {
	case TypeTraffic:
		return g.renderTraffic(seconds, title)
	case TypeErrors:
		return g.renderErrors(seconds, title)
	case TypeSPF:
		return g.renderSPF(seconds, title)
	case TypeDMARC:
		return g.renderDMARC(seconds, title)
	case TypeDKIM:
		return g.renderDKIM(seconds, title)
	case TypeDovecot:
		return g.renderDovecot(seconds, title)
	default:
		return "", fmt.Errorf("invalid chart type %q", chartType)
	}
}

func (g *Generator) renderTraffic(seconds int, title string) (string, error) {
	points, err := g.store.Fetch(g.store.MailPath(), seconds)
	if err != nil {
		if isMissingRRD(err) {
			return emptyChart(title, "msgs/min", "No data yet"), nil
		}
		return "", err
	}

	labels, series := buildSeries(points, "sent", "recv")
	line := baseLine(title, "msgs/min", labels)
	line.AddSeries("Sent", series[0],
		charts.WithAreaStyleOpts(opts.AreaStyle{Color: colors["sent"]}),
		charts.WithLineStyleOpts(opts.LineStyle{Color: colors["sent"]}),
	)
	line.AddSeries("Received", series[1],
		charts.WithLineStyleOpts(opts.LineStyle{Color: colors["recv"], Width: 2}),
	)
	return renderChart(line)
}

func (g *Generator) renderErrors(seconds int, title string) (string, error) {
	mailPts, mailErr := g.store.Fetch(g.store.MailPath(), seconds)
	virusPts, virusErr := g.store.Fetch(g.store.VirusPath(), seconds)
	if isMissingRRD(mailErr) && isMissingRRD(virusErr) {
		return emptyChart(title+" - Errors", "msgs/min", "No data yet"), nil
	}
	if mailErr != nil && !isMissingRRD(mailErr) {
		return "", mailErr
	}
	if virusErr != nil && !isMissingRRD(virusErr) {
		return "", virusErr
	}

	labels := labelsFrom(mailPts)
	if len(labels) == 0 {
		labels = labelsFrom(virusPts)
	}

	line := baseLine(title+" - Errors", "msgs/min", labels)
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

func (g *Generator) renderSPF(seconds int, title string) (string, error) {
	return g.renderTriple(g.store.MailPath(), seconds, title+" - SPF", "msgs/min", "spfpass", "spfnone", "spffail", "SPF pass", "SPF none", "SPF fail")
}

func (g *Generator) renderDMARC(seconds int, title string) (string, error) {
	return g.renderTriple(g.store.MailPath(), seconds, title+" - DMARC", "msgs/min", "dmarcpass", "dmarcnone", "dmarcfail", "DMARC pass", "DMARC none", "DMARC fail")
}

func (g *Generator) renderDKIM(seconds int, title string) (string, error) {
	return g.renderTriple(g.store.MailPath(), seconds, title+" - DKIM", "msgs/min", "dkimpass", "dkimnone", "dkimfail", "DKIM pass", "DKIM none", "DKIM fail")
}

func (g *Generator) renderDovecot(seconds int, title string) (string, error) {
	points, err := g.store.Fetch(g.store.DovecotPath(), seconds)
	if err != nil {
		if isMissingRRD(err) {
			return emptyChart(title+" - Dovecot", "logins/min", "No data yet"), nil
		}
		return "", err
	}

	labels, series := buildSeries(points, "dovecotloginsuccess", "dovecotloginfailed")
	line := baseLine(title+" - Dovecot", "logins/min", labels)
	line.AddSeries("Dovecot logins successful", series[0],
		charts.WithAreaStyleOpts(opts.AreaStyle{Color: colors["dovecotloginsuccess"]}),
		charts.WithLineStyleOpts(opts.LineStyle{Color: colors["dovecotloginsuccess"]}),
	)
	line.AddSeries("Dovecot logins failed", series[1],
		charts.WithLineStyleOpts(opts.LineStyle{Color: colors["dovecotloginfailed"], Width: 2}),
	)
	return renderChart(line)
}

func (g *Generator) renderTriple(path string, seconds int, title, yLabel, k1, k2, k3, n1, n2, n3 string) (string, error) {
	points, err := g.store.Fetch(path, seconds)
	if err != nil {
		if isMissingRRD(err) {
			return emptyChart(title, yLabel, "No data yet"), nil
		}
		return "", err
	}

	labels, series := buildSeries(points, k1, k2, k3)
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
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Width: "900px", Height: "200px"}),
		charts.WithTitleOpts(opts.Title{Title: title, Left: "center"}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true), Top: "bottom"}),
		charts.WithGridOpts(opts.Grid{Left: "3%", Right: "4%", Bottom: "15%", ContainLabel: opts.Bool(true)}),
		charts.WithXAxisOpts(opts.XAxis{
			Type: "category",
			Data: labels,
			AxisLabel: &opts.AxisLabel{Rotate: 45},
		}),
		charts.WithYAxisOpts(opts.YAxis{Name: yLabel, Type: "value", Min: opts.Float(0)}),
	)
	return line
}

func buildSeries(points []rrd.DataPoint, keys ...string) ([]string, [][]opts.LineData) {
	labels := make([]string, len(points))
	series := make([][]opts.LineData, len(keys))
	for i := range keys {
		series[i] = make([]opts.LineData, len(points))
	}
	for i, p := range points {
		labels[i] = p.Timestamp.Format("01-02 15:04")
		for j, key := range keys {
			series[j][i] = opts.LineData{Value: rrd.RatePerMinute(p.Values[key])}
		}
	}
	return labels, series
}

func labelsFrom(points []rrd.DataPoint) []string {
	labels := make([]string, len(points))
	for i, p := range points {
		labels[i] = p.Timestamp.Format("01-02 15:04")
	}
	return labels
}

func valuesFor(points []rrd.DataPoint, key string) []opts.LineData {
	data := make([]opts.LineData, len(points))
	for i, p := range points {
		data[i] = opts.LineData{Value: rrd.RatePerMinute(p.Values[key])}
	}
	return data
}

func renderChart(line *charts.Line) (string, error) {
	var buf bytes.Buffer
	if err := line.Render(io.MultiWriter(&buf)); err != nil {
		return "", err
	}
	return buf.String(), nil
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
	return buf.String()
}

func isMissingRRD(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "No such file") || strings.Contains(err.Error(), "not found")
}