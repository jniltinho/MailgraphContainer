# Mailgraph (Go)

**See what happens on your mail server — in charts.** Sent and received mail, spam, viruses, SPF, DMARC, DKIM, and more.

A modern **Golang** port of [Mailgraph](https://mailgraph.schweikert.ch). One binary replaces Perl + Apache + CGI and serves **interactive** charts in the browser.

| | |
|---|---|
| **Full documentation** | [docs/README.md](docs/README.md) — configuration, build, Docker, CLI |
| **Postfix on the server** | [README-POSTFIX.md](README-POSTFIX.md) — step-by-step install |
| **Docker image** | [Docker Hub — davidullrich/mailgraph](https://hub.docker.com/r/davidullrich/mailgraph) |

---

## How it works

![How Mailgraph works — from Postfix logs to browser charts](docs/screenshots/mailgraph-architecture.png)

1. **Postfix** writes events to `mail.log`
2. **Mailgraph** reads the log and stores history in **RRD** files
3. You open the **browser** at `/today`, `/last-week`, and other period URLs

No Apache, Perl, or manual PNG graphs.

---

## Quick start (Docker)

```bash
docker pull davidullrich/mailgraph:latest

docker run --rm -d \
  --name mailgraph \
  -v /var/log/mail/mail.log:/var/log/mail/mail.log:ro \
  -v /var/data/mailgraph/rrd:/var/www/mailgraph/rrd \
  -e MAILGRAPH_SERVER_HOSTNAME=mail.yourdomain.com \
  -p 8080:8080 \
  davidullrich/mailgraph:latest
```

Open **http://localhost:8080/today** — on first run, existing log history is imported automatically.

---

## Highlights

| | |
|---|---|
| **Image** | Alpine ~31 MB |
| **Binary** | ~3.2 MB (UPX) |
| **Charts** | Sent/received, errors, SPF, DMARC, DKIM, Dovecot, virus/spam |
| **Security** | Optional HTTPS and HTTP Basic Auth |
| **Compatibility** | Legacy Mailgraph RRD files, Postfix and other MTAs |

Includes the SPF / DMARC / DKIM patch by [Sebastian van de Meer](https://www.kernel-error.de/2014/04/22/mailgraph-graphen-um-spf-dmarc-und-dkim-erweitern/).

---

## Before vs now

| Before | Now |
|--------|-----|
| Perl + Apache + CGI | Single Go binary + [Echo v5](https://echo.labstack.com/) |
| Static PNG graphs | Interactive [go-echarts](https://github.com/go-echarts/go-echarts) charts |
| Heavy Debian image | **Alpine ~31 MB** |

---

## License

GNU General Public License v2 — see `backups/mailgraph/COPYING`.

**Credits:** [Mailgraph](https://mailgraph.schweikert.ch) (David Schweikert) · SPF/DMARC/DKIM patch (Sebastian van de Meer) · [Original Docker](https://www.production-ready.de/2023/04/15/mailgraph-docker-container-en.html) (David Ullrich) · Go port ([Go-Mailgraph](https://github.com/jniltinho/Go-Mailgraph))