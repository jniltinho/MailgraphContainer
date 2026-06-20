# Mailgraph on a Postfix server

Guide to install and run Mailgraph (Go edition) on the same host as **Postfix**, or via Docker with the local log mounted in.

Mailgraph reads the mail log, stores statistics in RRD files, and serves interactive charts at the web root (`http://<server>:8080/today` or `https://<server>:8443/today` with TLS). The root `/` redirects to `/today`.

---

## Requirements

| Component | Version / detail |
|-----------|------------------|
| Postfix | logs in syslog format |
| Go (build only) | 1.26+ |
| rrdtool | 1.7+ (required at runtime) |
| OS | Debian / Ubuntu (recommended), or any distro with Postfix + rrdtool |

Supported charts (when the respective services write logs):

- sent / received / rejected / bounced
- SPF (`policyd-spf`)
- DMARC (`opendmarc`)
- DKIM (`opendkim`)
- virus / spam (Amavis, ClamAV, SpamAssassin, etc.)
- Dovecot logins (if Dovecot runs on the same host)

---

## 1. Prepare the Postfix log

Mailgraph needs a readable log file with Postfix entries (and optionally Dovecot, Amavis, OpenDKIM, etc.).

### Debian / Ubuntu with rsyslog

Create `/etc/rsyslog.d/mailgraph.conf`:

```
# Dedicated log for Mailgraph
$template MailgraphFormat,"%TIMESTAMP% %HOSTNAME% %syslogtag%%msg%\n"

if $programname startswith 'postfix'
   or $programname == 'policyd-spf'
   or $programname == 'opendmarc'
   or $programname == 'opendkim'
   or $programname == 'dovecot'
   or $programname startswith 'amavis'
then {
    /var/log/mail/mail.log;MailgraphFormat
    stop
}
```

Apply and verify:

```bash
sudo mkdir -p /var/log/mail
sudo chown syslog:adm /var/log/mail
sudo chmod 750 /var/log/mail
sudo systemctl restart rsyslog

# Postfix lines should appear after sending/receiving a test email
sudo tail -f /var/log/mail/mail.log
```

Expected line example:

```
Jun 20 10:00:01 mail.example.com postfix/smtpd[1234]: ABCD: client=unknown[203.0.113.10]
```

### Alternative log paths

| Environment | Common path |
|-------------|-------------|
| Dedicated rsyslog | `/var/log/mail/mail.log` |
| General syslog | `/var/log/syslog` or `/var/log/messages` |
| journald only | configure rsyslog or redirect to a file (recommended) |

If you use another path, set `log.file` in `config.toml` or pass `--logfile` to the `server` subcommand.

---

## 2. Native binary installation

### 2.1 Dependencies

```bash
sudo apt update
sudo apt install -y rrdtool git
```

### 2.2 Build

```bash
git clone https://github.com/jniltinho/Go-Mailgraph.git
cd Go-Mailgraph

make build
# or: go build -trimpath -ldflags="-s -w" -o mailgraph .

sudo install -m 755 bin/mailgraph /usr/local/bin/mailgraph
```

### 2.3 Data directories

```bash
sudo mkdir -p /var/lib/mailgraph/rrd
sudo chown mailgraph:mailgraph /var/lib/mailgraph/rrd 2>/dev/null || sudo chown root:root /var/lib/mailgraph/rrd
```

On first run, if no RRD exists yet, the current history in `/var/log/mail/mail.log` is imported automatically.

### 2.4 Configuration file (recommended)

```bash
sudo mkdir -p /etc/mailgraph
sudo mailgraph generate-config
sudo cp config_*.toml /etc/mailgraph/config.toml
sudo nano /etc/mailgraph/config.toml
```

Example for production Postfix:

```toml
[log]
file = "/var/log/mail/mail.log"
type = "syslog"
year = 2026

[rrd]
dir = "/var/lib/mailgraph/rrd"
name = "mailgraph"

[server]
listen = "127.0.0.1:8080"
hostname = "mail.example.com"
tls_enabled = false
tls_cert = ""
tls_key = ""

[auth]
enabled = true
username = "admin"
password = "secret"
realm = "Mailgraph"

[filter]
ignore_localhost = true
```

Priority: flags > `MAILGRAPH_*` > `config.toml` > defaults. See [docs/README.md](docs/README.md#configuration) for the full list.

---

## 3. Manual run (testing)

```bash
sudo mailgraph server \
  --logfile=/var/log/mail/mail.log \
  --daemon-rrd=/var/lib/mailgraph/rrd \
  --hostname=$(hostname -f) \
  --listen=127.0.0.1:8080
```

With `config.toml` in `/etc/mailgraph/`:

```bash
sudo mailgraph server
```

Open in the browser (via SSH tunnel or reverse proxy):

```
http://127.0.0.1:8080/today
```

Available periods (each with its own URL):

| Period | URL |
|--------|-----|
| Today (current day since 00:00) | `/today` |
| Last Day (rolling 24 h) | `/last-day` |
| Last Week | `/last-week` |
| Last 2 Weeks | `/last-2-weeks` |
| Last Month | `/last-month` |
| Last 2 Month | `/last-2-month` |
| Last Year | `/last-year` |
| Last 2 Years | `/last-2-years` |

### Import historical log without starting the web server

```bash
sudo mailgraph cat \
  --logfile=/var/log/mail/mail.log \
  --daemon-rrd=/var/lib/mailgraph/rrd \
  --year=$(date +%Y) \
  --verbose
```

### Useful Postfix options

| Flag / config | When to use |
|---------------|-------------|
| `--ignore-localhost` / `filter.ignore_localhost` | Ignore traffic to/from `127.0.0.1` (local scanners, Amavis on loopback) |
| `--ignore-host=HOST` / `filter.ignore_hosts` | Ignore relay from a specific host (regex, repeatable) |
| `--rbl-is-spam` / `filter.rbl_is_spam` | Count RBL rejections as spam |
| `--virbl-is-virus` / `filter.virbl_is_virus` | Count VIRBL rejections as virus |
| `--host=mail.example.com` / `log.host_filter` | Filter syslog entries by hostname only |
| `--listen=127.0.0.1:8080` / `server.listen` | Listen on localhost only (more secure) |
| `--tls` / `server.tls_enabled` | Enable HTTPS with a PEM certificate |
| `--tls-cert` / `server.tls_cert` | TLS certificate path |
| `--tls-key` / `server.tls_key` | TLS private key path |
| `--auth` / `auth.enabled` | Enable HTTP Basic Auth |
| `--auth-user` / `auth.username` | Auth username |
| `--auth-pass` / `auth.password` | Auth password |
| `--auth-realm` / `auth.realm` | Login prompt realm |

### HTTP Basic Auth

Protects the web UI with Echo's built-in authentication:

```toml
[auth]
enabled = true
username = "admin"
password = "secret"
realm = "Mailgraph"
```

```bash
sudo mailgraph server --config /etc/mailgraph/config.toml
```

Combine with TLS when exposing the service publicly.

### HTTPS with TLS

For local testing:

```bash
make certs
sudo mailgraph server \
  --listen=:8443 \
  --tls \
  --tls-cert=ssl/server.crt \
  --tls-key=ssl/server.key
```

Example with a Let's Encrypt certificate:

```toml
[server]
listen = ":8443"
hostname = "mail.example.com"
tls_enabled = true
tls_cert = "/etc/letsencrypt/live/mail.example.com/fullchain.pem"
tls_key = "/etc/letsencrypt/live/mail.example.com/privkey.pem"
```

```bash
sudo mailgraph server --config /etc/mailgraph/config.toml
```

Access: `https://mail.example.com:8443/today`

Example with Amavis on localhost:

```bash
sudo mailgraph server \
  --logfile=/var/log/mail/mail.log \
  --daemon-rrd=/var/lib/mailgraph/rrd \
  --ignore-localhost \
  --hostname=mail.example.com \
  --listen=127.0.0.1:8080
```

---

## 4. systemd service (production)

Create `/etc/systemd/system/mailgraph.service`:

```ini
[Unit]
Description=Mailgraph mail statistics
After=network.target rsyslog.service postfix.service
Wants=rsyslog.service

[Service]
Type=simple
User=root
Group=root
ExecStart=/usr/local/bin/mailgraph server \
  --config /etc/mailgraph/config.toml
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Alternative without a config file (inline flags):

```ini
ExecStart=/usr/local/bin/mailgraph server \
  --logfile=/var/log/mail/mail.log \
  --daemon-rrd=/var/lib/mailgraph/rrd \
  --hostname=mail.example.com \
  --ignore-localhost \
  --listen=127.0.0.1:8080
```

With TLS in systemd (certificate on the host):

```ini
ExecStart=/usr/local/bin/mailgraph server \
  --logfile=/var/log/mail/mail.log \
  --daemon-rrd=/var/lib/mailgraph/rrd \
  --hostname=mail.example.com \
  --listen=:8443 \
  --tls \
  --tls-cert=/etc/letsencrypt/live/mail.example.com/fullchain.pem \
  --tls-key=/etc/letsencrypt/live/mail.example.com/privkey.pem
```

For public exposure, use `config.toml` with TLS and `[auth]` enabled (section 2.4).

Replace `mail.example.com` with your server's FQDN.

Enable the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now mailgraph
sudo systemctl status mailgraph
```

Service logs:

```bash
sudo journalctl -u mailgraph -f
```

---

## 5. Docker on the same Postfix host

If Postfix already runs on the host, mount the log and RRD directory:

```bash
sudo mkdir -p /var/lib/mailgraph/rrd

docker run --rm -d \
  --name mailgraph \
  --restart unless-stopped \
  -v /var/log/mail/mail.log:/var/log/mail/mail.log:ro \
  -v /var/lib/mailgraph/rrd:/var/www/mailgraph/rrd \
  -v /etc/localtime:/etc/localtime:ro \
  -p 127.0.0.1:8080:8080 \
  davidullrich/mailgraph:latest
```

The container entrypoint runs `mailgraph server` automatically. To override:

```bash
docker run --rm -d \
  --name mailgraph \
  -v /var/log/mail/mail.log:/var/log/mail/mail.log:ro \
  -v /var/lib/mailgraph/rrd:/var/www/mailgraph/rrd \
  -e MAILGRAPH_SERVER_HOSTNAME=mail.example.com \
  -e MAILGRAPH_FILTER_IGNORE_LOCALHOST=true \
  -p 127.0.0.1:8080:8080 \
  davidullrich/mailgraph:latest
```

Charts at `http://127.0.0.1:8080/today`.

With TLS in Docker:

```bash
docker run --rm -d \
  --name mailgraph \
  --restart unless-stopped \
  -v /var/log/mail/mail.log:/var/log/mail/mail.log:ro \
  -v /var/lib/mailgraph/rrd:/var/www/mailgraph/rrd \
  -v /etc/letsencrypt/live/mail.example.com/fullchain.pem:/etc/ssl/certs/mailgraph.crt:ro \
  -v /etc/letsencrypt/live/mail.example.com/privkey.pem:/etc/ssl/private/mailgraph.key:ro \
  -e MAILGRAPH_SERVER_LISTEN=:8443 \
  -e MAILGRAPH_SERVER_TLS_ENABLED=true \
  -e MAILGRAPH_SERVER_TLS_CERT=/etc/ssl/certs/mailgraph.crt \
  -e MAILGRAPH_SERVER_TLS_KEY=/etc/ssl/private/mailgraph.key \
  -p 8443:8443 \
  davidullrich/mailgraph:latest
```

With Basic Auth in Docker:

```bash
docker run --rm -d \
  --name mailgraph \
  --restart unless-stopped \
  -v /var/log/mail/mail.log:/var/log/mail/mail.log:ro \
  -v /var/lib/mailgraph/rrd:/var/www/mailgraph/rrd \
  -e MAILGRAPH_AUTH_ENABLED=true \
  -e MAILGRAPH_AUTH_USERNAME=admin \
  -e MAILGRAPH_AUTH_PASSWORD=secret \
  -p 127.0.0.1:8080:8080 \
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
      - /var/lib/mailgraph/rrd:/var/www/mailgraph/rrd
      - /etc/localtime:/etc/localtime:ro
    ports:
      - "127.0.0.1:8080:8080"
```

---

## 6. Verification

```bash
# Service running
systemctl is-active mailgraph

# Postfix log is being written
sudo tail -5 /var/log/mail/mail.log

# RRD files created/updated
ls -la /var/lib/mailgraph/rrd/
# mailgraph.rrd  mailgraph_virus.rrd  mailgraph_dovecot.rrd

# Web UI (HTTP)
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:8080/
# Expected: 302 (redirect to /today) or 401 if auth is enabled

# Web UI with Basic Auth
curl -s -o /dev/null -w "%{http_code}\n" -u admin:secret http://127.0.0.1:8080/today
# Expected: 200

# Web UI (HTTPS, if TLS enabled)
curl -sk -o /dev/null -w "%{http_code}\n" https://127.0.0.1:8443/today
# Expected: 200
```

Send a test email (inbound and outbound) and wait 1–2 minutes; charts refresh automatically every 5 minutes on the page.

---

## 7. Troubleshooting

### Empty charts

1. Confirm `/var/log/mail/mail.log` receives `postfix/...` lines
2. Check read permissions for the user running Mailgraph
3. Process the log manually with `mailgraph cat --verbose` and watch for `rrdtool` errors
4. Confirm `rrdtool` is installed: `which rrdtool`

### RRD stopped updating

- Log timestamps must not go backwards (clock skew or wrong year → use `--year` or `log.year`)
- Inspect the latest timestamp: `rrdtool last /var/lib/mailgraph/rrd/mailgraph.rrd`

### Only sent traffic, nothing received

- Fetchmail or a local relay may use `127.0.0.1` → use `--ignore-localhost` or adjust fetchmail's `smtphost`

### No SPF / DKIM / DMARC data

- The log must contain entries from `policyd-spf`, `opendkim`, and `opendmarc`
- Include those programs in the rsyslog filter (section 1)

### Port exposed to the internet

- Prefer `server.listen = "127.0.0.1:8080"` and access via SSH tunnel or VPN
- If exposing publicly, use TLS (`server.tls_enabled = true`) with a valid certificate
- Also enable `auth.enabled = true` with a strong username and password
- Do not expose mail statistics publicly without protection

### TLS startup error

- Confirm `tls_cert` and `tls_key` exist and are readable by the service user
- Certificate and key must be in PEM format
- `tls_enabled = true` requires both paths to be set

### 401 error on the web UI

- `auth.enabled = true` requires credentials in the browser or `curl -u user:pass`
- Confirm `auth.username` and `auth.password` in `config.toml` or `MAILGRAPH_AUTH_*` variables

---

## 8. Quick command reference

```bash
# Help
mailgraph --help
mailgraph server --help

# Version
mailgraph version

# Run in foreground (debug)
sudo mailgraph server --verbose \
  --logfile=/var/log/mail/mail.log \
  --daemon-rrd=/var/lib/mailgraph/rrd \
  --listen=127.0.0.1:8080

# Reprocess the full log (no web server)
sudo mailgraph cat \
  --logfile=/var/log/mail/mail.log \
  --daemon-rrd=/var/lib/mailgraph/rrd \
  --verbose

# HTTP Basic Auth
sudo mailgraph server \
  --auth \
  --auth-user=admin \
  --auth-pass=secret \
  --listen=127.0.0.1:8080

# HTTPS with TLS
sudo mailgraph server \
  --listen=:8443 \
  --tls \
  --tls-cert=/etc/letsencrypt/live/mail.example.com/fullchain.pem \
  --tls-key=/etc/letsencrypt/live/mail.example.com/privkey.pem

# Generate config.toml
mailgraph generate-config
```

---

## Links

- [Original Mailgraph](https://mailgraph.schweikert.ch/)
- [Project README](README.md) — overview and quick start
- [Full documentation](docs/README.md) — configuration, build, Docker
- SPF/DMARC/DKIM patch: [kernel-error.de](https://www.kernel-error.de/2014/04/22/mailgraph-graphen-um-spf-dmarc-und-dkim-erweitern/)