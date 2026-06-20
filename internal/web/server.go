// Package web serves the mailgraph HTML UI and chart endpoints with Echo.
package web

import (
	"crypto/subtle"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"mailgraph/internal/buildinfo"
	"mailgraph/internal/charts"
	"mailgraph/internal/config"
	"mailgraph/internal/rrd"
)

const periodPageTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>Mail statistics for {{.Hostname}} - {{.CurrentTitle}}</title>
  <meta http-equiv="refresh" content="300" />
  <meta http-equiv="pragma" content="no-cache" />
  <link rel="stylesheet" href="/mailgraph.css" type="text/css" />
</head>
<body>
  <h1>Mail statistics for {{.Hostname}}</h1>
  <ul id="jump">
  {{range .Periods}}<li><a href="/{{.Slug}}"{{if eq .Index $.CurrentIndex}} class="active"{{end}}>{{.Title}}</a>&nbsp;</li>{{end}}
  </ul>
  <h2>{{.CurrentTitle}}</h2>
  <div class="charts">
    <iframe class="chart" src="/chart?period={{.CurrentIndex}}&amp;type=n" title="{{.CurrentTitle}} - traffic" loading="lazy" scrolling="no"></iframe>
    <iframe class="chart" src="/chart?period={{.CurrentIndex}}&amp;type=e" title="{{.CurrentTitle}} - errors" loading="lazy" scrolling="no"></iframe>
    <iframe class="chart" src="/chart?period={{.CurrentIndex}}&amp;type=s" title="{{.CurrentTitle}} - SPF" loading="lazy" scrolling="no"></iframe>
    <iframe class="chart" src="/chart?period={{.CurrentIndex}}&amp;type=d" title="{{.CurrentTitle}} - DMARC" loading="lazy" scrolling="no"></iframe>
    <iframe class="chart" src="/chart?period={{.CurrentIndex}}&amp;type=k" title="{{.CurrentTitle}} - DKIM" loading="lazy" scrolling="no"></iframe>
    <iframe class="chart" src="/chart?period={{.CurrentIndex}}&amp;type=v" title="{{.CurrentTitle}} - Dovecot" loading="lazy" scrolling="no"></iframe>
  </div>
  <hr/>
  <a href="https://mailgraph.schweikert.ch/">Mailgraph</a> {{.Version}} (Go port)
</body>
</html>`

type periodPageData struct {
	Hostname     string
	Version      string
	CurrentIndex int
	CurrentTitle string
	Periods      []struct {
		Title string
		Slug  string
		Index int
	}
}

// Server handles HTTP routes for the mailgraph dashboard.
type Server struct {
	cfg          config.Config
	store        *rrd.Store
	charts       *charts.Generator
	templates    *template.Template
	mailgraphCSS []byte
}

// New creates a Server from cfg and embedded static assets.
func New(cfg config.Config, mailgraphCSS []byte) *Server {
	store := rrd.NewStore(cfg.RRDDir, cfg.RRDName, cfg.OnlyMailRRD, cfg.OnlyVirusRRD, false)
	tmpl := template.Must(template.New("period").Parse(periodPageTmpl))
	return &Server{
		cfg:          cfg,
		store:        store,
		charts:       charts.NewGenerator(store),
		templates:    tmpl,
		mailgraphCSS: mailgraphCSS,
	}
}

// Register mounts routes on e, including optional Basic Auth middleware.
func (s *Server) Register(e *echo.Echo) {
	if s.cfg.AuthEnabled {
		username := s.cfg.AuthUsername
		password := s.cfg.AuthPassword
		realm := s.cfg.AuthRealm
		if realm == "" {
			realm = "Mailgraph"
		}

		e.Use(middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
			Realm: realm,
			Validator: func(c *echo.Context, user, pass string) (bool, error) {
				userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(username)) == 1
				passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(password)) == 1
				return userMatch && passMatch, nil
			},
		}))
	}

	e.GET("/", s.home)
	for i, p := range charts.Periods {
		e.GET("/"+p.Slug, s.periodPage(i))
	}
	e.GET("/chart", s.chart)
	e.GET("/mailgraph.css", s.css)
}

func (s *Server) css(c *echo.Context) error {
	c.Response().Header().Set("Content-Type", "text/css; charset=utf-8")
	_, err := c.Response().Write(s.mailgraphCSS)
	return err
}

func (s *Server) home(c *echo.Context) error {
	return c.Redirect(http.StatusFound, "/"+charts.Periods[0].Slug)
}

func (s *Server) periodPage(index int) echo.HandlerFunc {
	return func(c *echo.Context) error {
		return s.renderPeriodPage(c, index)
	}
}

func (s *Server) renderPeriodPage(c *echo.Context, index int) error {
	if index < 0 || index >= len(charts.Periods) {
		return echo.ErrNotFound
	}

	data := periodPageData{
		Hostname:     s.cfg.Hostname,
		Version:      buildinfo.Version,
		CurrentIndex: index,
		CurrentTitle: charts.Periods[index].Title,
	}

	for i, p := range charts.Periods {
		data.Periods = append(data.Periods, struct {
			Title string
			Slug  string
			Index int
		}{Title: p.Title, Slug: p.Slug, Index: i})
	}

	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	return s.templates.Execute(c.Response(), data)
}

func (s *Server) chart(c *echo.Context) error {
	periodStr := c.QueryParam("period")
	chartType := c.QueryParam("type")

	if chartType == "" {
		// legacy query format: 0-n, 1-e, etc.
		q := c.QueryParam("q")
		if q == "" {
			q = c.Request().URL.RawQuery
		}
		if idx := strings.Index(q, "-"); idx > 0 {
			periodStr = q[:idx]
			chartType = q[idx+1:]
		}
	}

	period, err := strconv.Atoi(periodStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid period")
	}

	html, err := s.charts.Render(period, chartType)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("chart error: %v", err))
	}

	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err = c.Response().Write([]byte(html))
	return err
}