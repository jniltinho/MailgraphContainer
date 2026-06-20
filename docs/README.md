# Mailgraph — documentation

Detailed reference for configuration, deployment, development, and the web interface.

**Quick links:** [Project README](../README.md) · [Postfix install](../README-POSTFIX.md) · [Docker Hub](https://hub.docker.com/r/davidullrich/mailgraph)

---

## Who is it for?

- **Postfix** administrators who want to understand mail traffic on their server
- Teams migrating from classic Mailgraph who want a **lighter, modern** stack
- Anyone who prefers **Docker** and a quick setup
- Providers and companies that need optional **HTTPS** and **password-protected** dashboards

---

## Architecture

![How Mailgraph works — from Postfix logs to browser charts](screenshots/mailgraph-architecture.png)

| Step | Interval |
|------|----------|
| Log reading | Real time (`tail -f`) |
| RRD writes | **1-minute** buckets |
| Page refresh | **5 minutes** (meta refresh) |

On first run without existing RRD files, the full log history is imported automatically.

---

## Dashboard

Interactive charts for:

- **Sent** and **received** messages
- **Rejected**, bounced, **virus**, and **spam**
- **SPF**, **DMARC**, and **DKIM** results
- **Dovecot** logins (when applicable)

### Time periods

| Period | URL | Meaning |
|--------|-----|---------|
| **Today** | `/today` | Today since 00:00 (local time) |
| Last Day | `/last-day` | Rolling last 24 hours |
| Last Week | `/last-week` | Last 7 days |
| Last 2 Weeks | `/last-2-weeks` | Last 14 days |
| Last Month | `/last-month` | Last 31 days |
| Last 2 Month | `/last-2-month` | Last 62 days |
| Last Year | `/last-year` | Last 365 days |
| Last 2 Years | `/last-2-years` | Last 730 days |

The root `/` redirects to `/today`.

### Web routes

| Route | Description |
|-------|-------------|
| `/today`, `/last-day`, … | Page with 6 charts for the period |
| `/mailgraph.css` | CSS embedded in the binary |
| `/chart?period=N&type=T` | Single chart HTML (`T` = `n`/`e`/`s`/`d`/`k`/`v`) |

### Horizontal axis labels

All labels use the **server local timezone**. Formats follow the legacy Perl `mailgraph.cgi` / `rrdtool graph` behaviour where possible.

| Period | Axis label format | Example | Notes |
|--------|-------------------|---------|-------|
| Today, Last Day | hour:minute | `19:48` | One label per RRD sample (~8 min) |
| Last Week, Last 2 Weeks | weekday + day + month | `Mon 15 Jun` | **One label per calendar day** (like `rrdtool` PNG graphs) |
| Last Month, Last 2 Month | month-day | `06-15` | |
| Last Year, Last 2 Years | full date | `2026-06-15` | |

#### Legacy Perl (`mailgraph.cgi`)

The original CGI did **not** set axis formats in Perl code. It called `RRDs::graph` with `--start -$range` and let **rrdtool** choose the labels. For **Last Week**, the PNG graphs show one tick per day, for example:

`Sun 14 Jun` · `Mon 15 Jun` · `Tue 16 Jun` · … · `Sat 20 Jun`

The Go port uses the same weekday + date + month style for week views. Tooltips on every sample still show the exact time when you hover a point.

---

## Features

- Real-time log reading (`tail -f`)
- Compatible with legacy Mailgraph RRD files
- Dedicated URL per period (share links like `/last-week`)
- Optional HTTPS and HTTP Basic Auth
- Web UI at the root (`/`) — no `/mailgraph` prefix
- CSS embedded in the binary (`go:embed` in `main.go`)
- Postfix, Sendmail, Exim, Amavis, ClamAV, SpamAssassin, and more

---

## Tech stack

- **Go** 1.26 · **Cobra** + **Viper** · **Echo** v5 · **go-echarts** v2 · **rrdtool** · **UPX**

## Project layout

```
main.go                 # entrypoint; go:embed web/static/mailgraph.css
cmd/                    # CLI (server, cat, version, generate-config)
web/static/             # CSS and assets (embedded in the binary)
internal/               # collector, rrd, charts, web, config…
docs/screenshots/       # documentation diagrams and images
Dockerfile              # multi-stage: Go + UPX → Alpine
docker-compose.test.yml # local test container on port 8585
Makefile                # build, Docker, remote log fetch
```

---

## CLI

```bash
mailgraph server           # collector + HTTP server (Docker default)
mailgraph cat              # process the log once and exit
mailgraph version          # version and build info
mailgraph generate-config  # generate config.toml from the embedded template
mailgraph --help
mailgraph server --help
```

In the container, `entrypoint.sh` runs `mailgraph server` by default. Arguments passed to `docker run` override that behavior.

---

## Configuration

Priority: **flags** > **`MAILGRAPH_*` env vars** > **`config.toml`** > **defaults**.

Config file search paths: `./config.toml`, `/etc/mailgraph/config.toml`, `~/.mailgraph/config.toml`

Use `--config /path/config.toml` for a specific file.

### Minimal example (`config.toml`)

```toml
[log]
file = "/var/log/mail/mail.log"
type = "syslog"
year = 2026

[rrd]
dir = "/var/lib/mailgraph/rrd"
name = "mailgraph"

[server]
listen = ":8080"
hostname = "mail.example.com"

[filter]
ignore_localhost = true
```

Generate a starter file with `mailgraph generate-config` or copy `config.toml.example`.

### HTTP Basic Auth

```toml
[auth]
enabled = true
username = "admin"
password = "secret"
realm = "Mailgraph"
```

```bash
mailgraph server --auth --auth-user=admin --auth-pass=secret
```

### HTTPS (TLS)

For local testing:

```bash
make certs   # ssl/server.crt and ssl/server.key

mailgraph server --listen=:8443 --tls \
  --tls-cert=ssl/server.crt --tls-key=ssl/server.key
```

Charts at **https://localhost:8443/today**

Production example:

```toml
[server]
listen = ":8443"
tls_enabled = true
tls_cert = "/etc/ssl/certs/mailgraph.crt"
tls_key = "/etc/ssl/private/mailgraph.key"
```

### Environment variables

| Variable | Equivalent |
|----------|------------|
| `MAILGRAPH_LOG_FILE` | `log.file` |
| `MAILGRAPH_LOG_TYPE` | `log.type` |
| `MAILGRAPH_LOG_YEAR` | `log.year` |
| `MAILGRAPH_RRD_DIR` | `rrd.dir` |
| `MAILGRAPH_SERVER_LISTEN` | `server.listen` |
| `MAILGRAPH_SERVER_HOSTNAME` | `server.hostname` |
| `MAILGRAPH_SERVER_TLS_ENABLED` | `server.tls_enabled` |
| `MAILGRAPH_SERVER_TLS_CERT` | `server.tls_cert` |
| `MAILGRAPH_SERVER_TLS_KEY` | `server.tls_key` |
| `MAILGRAPH_AUTH_ENABLED` | `auth.enabled` |
| `MAILGRAPH_AUTH_USERNAME` | `auth.username` |
| `MAILGRAPH_AUTH_PASSWORD` | `auth.password` |
| `MAILGRAPH_AUTH_REALM` | `auth.realm` |
| `MAILGRAPH_FILTER_IGNORE_LOCALHOST` | `filter.ignore_localhost` |
| `MAILGRAPH_APP_VERBOSE` | `app.verbose` |

### Main flags

| Flag | Description |
|------|-------------|
| `--logfile` | Syslog mail log file |
| `--daemon-rrd` | Directory for `.rrd` files |
| `--hostname` | Name shown in chart titles |
| `--listen` | Listen address (default `:8080`) |
| `--tls` / `--tls-cert` / `--tls-key` | HTTPS |
| `--auth` / `--auth-user` / `--auth-pass` | Basic Auth |
| `--ignore-localhost` | Ignore traffic to/from `127.0.0.1` |
| `--ignore-host` | Ignore host (regex, repeatable) |
| `--verbose` | Verbose output |
| `--daemon` | Write PID file and detach |

---

## Build

### Requirements

- Go 1.26+
- `rrdtool` (runtime)
- `make`
- UPX (optional, for `build-prod` and Docker)

Module: `mailgraph` (repository root).

### Commands

```bash
make deps          # Go modules
make build         # bin/mailgraph (~11 MB)
make build-prod    # bin/mailgraph + UPX (~3.2 MB)
make build-docker  # Docker image
make run           # build + local server
make certs         # self-signed TLS certs in ssl/
make test          # go test ./...
make help
```

Production build with UPX:

```bash
make install-upx
make build-prod
./bin/mailgraph version
```

Custom image tag:

```bash
make build-docker IMAGE=your-user/mailgraph:latest
```

---

## Docker

### Basic run

```bash
make build-docker

docker run --rm -d \
  --name mailgraph \
  -v /var/log/mail/mail.log:/var/log/mail/mail.log:ro \
  -v /var/data/mailgraph/rrd:/var/www/mailgraph/rrd \
  -v /etc/localtime:/etc/localtime:ro \
  -p 8080:8080 \
  davidullrich/mailgraph:latest
```

Charts: **http://localhost:8080/today**

### With config file or environment

```bash
docker run --rm -d \
  --name mailgraph \
  -v /var/log/mail/mail.log:/var/log/mail/mail.log:ro \
  -v /var/data/mailgraph/rrd:/var/www/mailgraph/rrd \
  -v /etc/mailgraph/config.toml:/etc/mailgraph/config.toml:ro \
  -e MAILGRAPH_SERVER_HOSTNAME=mail.example.com \
  -p 8080:8080 \
  davidullrich/mailgraph:latest
```

### With TLS

```bash
docker run --rm -d --name mailgraph \
  -v /var/log/mail/mail.log:/var/log/mail/mail.log:ro \
  -v /var/data/mailgraph/rrd:/var/www/mailgraph/rrd \
  -v /etc/letsencrypt/live/mail.example.com/fullchain.pem:/etc/ssl/certs/mailgraph.crt:ro \
  -v /etc/letsencrypt/live/mail.example.com/privkey.pem:/etc/ssl/private/mailgraph.key:ro \
  -e MAILGRAPH_SERVER_LISTEN=:8443 \
  -e MAILGRAPH_SERVER_TLS_ENABLED=true \
  -e MAILGRAPH_SERVER_TLS_CERT=/etc/ssl/certs/mailgraph.crt \
  -e MAILGRAPH_SERVER_TLS_KEY=/etc/ssl/private/mailgraph.key \
  -p 8443:8443 \
  davidullrich/mailgraph:latest
```

### With Basic Auth

```bash
docker run --rm -d --name mailgraph \
  -v /var/log/mail/mail.log:/var/log/mail/mail.log:ro \
  -v /var/data/mailgraph/rrd:/var/www/mailgraph/rrd \
  -e MAILGRAPH_AUTH_ENABLED=true \
  -e MAILGRAPH_AUTH_USERNAME=admin \
  -e MAILGRAPH_AUTH_PASSWORD=secret \
  -p 8080:8080 \
  davidullrich/mailgraph:latest
```

### Docker Compose

```yaml
services:
  mailgraph:
    image: davidullrich/mailgraph:latest
    hostname: mail.example.com
    restart: unless-stopped
    volumes:
      - /var/log/mail/mail.log:/var/log/mail/mail.log:ro
      - /var/data/mailgraph/rrd:/var/www/mailgraph/rrd
      - /etc/localtime:/etc/localtime:ro
    ports:
      - "8080:8080"
```

### Local testing with a remote log

```bash
make fetch-testdata TESTDATA_HOST=mx01    # saves to testdata/mail.log (gitignored)
make test-docker                          # http://127.0.0.1:8585/today
make test-docker-validate
make test-docker-down
```

To reprocess the log from scratch: `rm -rf testdata/rrd/*` before starting the container.

---

## Screenshots

### Last week

![Last week](../screenshots/lastweek.png)

### Last month

![Last month](../screenshots/lastmonth.png)

---

## Credits

- [Mailgraph](https://mailgraph.schweikert.ch) — David Schweikert (GPL)
- SPF/DMARC/DKIM patch — Sebastian van de Meer
- Original Docker container — [David Ullrich](https://www.production-ready.de/2023/04/15/mailgraph-docker-container-en.html) ([DE](https://www.production-ready.de/2023/04/15/mailgraph-docker-container.html))
- Go port — [Go-Mailgraph](https://github.com/jniltinho/Go-Mailgraph)