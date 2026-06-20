package web

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/davidullrich/mailgraph/internal/charts"
	"github.com/davidullrich/mailgraph/internal/config"
	"github.com/davidullrich/mailgraph/internal/rrd"
)

//go:embed static/mailgraph.css
var mailgraphCSS []byte

type Server struct {
	cfg       config.Config
	store     *rrd.Store
	charts    *charts.Generator
	templates *template.Template
}

func New(cfg config.Config) *Server {
	store := rrd.NewStore(cfg.RRDDir, cfg.RRDName, cfg.OnlyMailRRD, cfg.OnlyVirusRRD, false)
	return &Server{
		cfg:    cfg,
		store:  store,
		charts: charts.NewGenerator(store),
	}
}

func (s *Server) Register(e *echo.Echo) {
	e.GET("/mailgraph/", s.index)
	e.GET("/mailgraph", s.index)
	e.GET("/mailgraph/chart", s.chart)
	e.GET("/mailgraph/mailgraph.css", s.css)
}

func (s *Server) css(c *echo.Context) error {
	c.Response().Header().Set("Content-Type", "text/css; charset=utf-8")
	_, err := c.Response().Write(mailgraphCSS)
	return err
}

func (s *Server) index(c *echo.Context) error {
	data := struct {
		Hostname string
		Version  string
		Periods  []struct {
			Title string
			Index int
		}
	}{
		Hostname: s.cfg.Hostname,
		Version:  config.Version,
	}

	for i, p := range charts.Periods {
		data.Periods = append(data.Periods, struct {
			Title string
			Index int
		}{Title: p.Title, Index: i})
	}

	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>Mail statistics for {{.Hostname}}</title>
  <meta http-equiv="refresh" content="300" />
  <meta http-equiv="pragma" content="no-cache" />
  <link rel="stylesheet" href="/mailgraph/mailgraph.css" type="text/css" />
</head>
<body>
  <h1>Mail statistics for {{.Hostname}}</h1>
  <ul id="jump">
  {{range .Periods}}<li><a href="#G{{.Index}}">{{.Title}}</a>&nbsp;</li>{{end}}
  </ul>
  {{range .Periods}}
  <h2 id="G{{.Index}}">{{.Title}}</h2>
  <p>
    <div class="chart" data-period="{{.Index}}" data-type="n"></div><br/>
    <div class="chart" data-period="{{.Index}}" data-type="e"></div><br/>
    <div class="chart" data-period="{{.Index}}" data-type="s"></div><br/>
    <div class="chart" data-period="{{.Index}}" data-type="d"></div><br/>
    <div class="chart" data-period="{{.Index}}" data-type="k"></div><br/>
    <div class="chart" data-period="{{.Index}}" data-type="v"></div><br/>
  </p>
  {{end}}
  <hr/>
  <a href="https://mailgraph.schweikert.ch/">Mailgraph</a> {{.Version}} (Go port)
  <script>
  document.querySelectorAll('.chart').forEach(function(el) {
    fetch('/mailgraph/chart?period=' + el.dataset.period + '&type=' + el.dataset.type)
      .then(function(r) { return r.text(); })
      .then(function(html) { el.innerHTML = html; });
  });
  </script>
</body>
</html>`

	t, err := template.New("index").Parse(tmpl)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	return t.Execute(c.Response(), data)
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