# Mailgraph (Go)

**Veja em gráficos o que acontece no seu servidor de e-mail** — envios, recebimentos, spam, vírus, SPF, DMARC, DKIM e muito mais.

Port moderno em **Golang** do [Mailgraph](https://mailgraph.schweikert.ch). Um único binário substitui Perl + Apache + CGI e entrega gráficos **interativos** no navegador.

| | |
|---|---|
| **Instalação Postfix** | [README-POSTFIX.md](README-POSTFIX.md) — passo a passo no servidor |
| **Imagem Docker** | [Docker Hub — davidullrich/mailgraph](https://hub.docker.com/r/davidullrich/mailgraph) |

---

## Como funciona (visão geral)

![Como o Mailgraph funciona — do log do Postfix aos gráficos no navegador](docs/screenshots/mailgraph-architecture.png)

1. O **Postfix** (ou outro MTA) grava eventos no arquivo de log (`mail.log`).
2. O **Mailgraph** lê esse log, conta os eventos e salva o histórico em arquivos **RRD**.
3. Você abre o **navegador** e vê gráficos por período: hoje, última semana, último mês, etc.

Não precisa instalar Apache, Perl nem gerar imagens PNG manualmente.

---

## Para quem é?

- Administradores de **Postfix** que querem entender o tráfego de e-mail do servidor
- Equipes que já usavam o Mailgraph clássico e querem a versão **mais leve e moderna**
- Quem prefere subir tudo com **Docker** em poucos minutos
- Provedores e empresas que precisam de **HTTPS** e **senha no painel** (opcional)

---

## Início rápido com Docker

O jeito mais simples de testar — só precisa do Docker e do arquivo de log do seu servidor de e-mail.

```bash
# 1. Baixe ou construa a imagem
docker pull davidullrich/mailgraph:latest
# ou: make build-docker

# 2. Suba o container (ajuste os caminhos do seu servidor)
docker run --rm -d \
  --name mailgraph \
  -v /var/log/mail/mail.log:/var/log/mail/mail.log:ro \
  -v /var/data/mailgraph/rrd:/var/www/mailgraph/rrd \
  -e MAILGRAPH_SERVER_HOSTNAME=mail.seudominio.com \
  -p 8080:8080 \
  davidullrich/mailgraph:latest

# 3. Abra no navegador
# http://localhost:8080/today
```

Na **primeira execução**, o Mailgraph importa o histórico do log automaticamente. Depois disso, acompanha novos eventos em tempo real.

| O que montar | Para quê |
|--------------|----------|
| `mail.log` (somente leitura) | Fonte dos dados — log do Postfix |
| pasta `rrd/` | Onde o histórico fica salvo (persistente) |

---

## O que você vê no painel

Gráficos interativos ([go-echarts](https://github.com/go-echarts/go-echarts)) com:

- Mensagens **enviadas** e **recebidas**
- **Rejeitadas**, bounced, **vírus** e **spam**
- Resultados **SPF**, **DMARC** e **DKIM**
- Logins **Dovecot** (se aplicável)

Cada período tem sua própria página:

| Período | URL | Significado |
|---------|-----|-------------|
| **Today** | `/today` | Hoje, desde 00:00 até agora |
| Last Day | `/last-day` | Últimas 24 horas corridas |
| Last Week | `/last-week` | Últimos 7 dias |
| Last 2 Weeks | `/last-2-weeks` | Últimas 2 semanas |
| Last Month | `/last-month` | Último mês |
| Last 2 Month | `/last-2-month` | Últimos 2 meses |
| Last Year | `/last-year` | Último ano |
| Last 2 Years | `/last-2-years` | Últimos 2 anos |

A raiz `/` redireciona para `/today`. A página recarrega a cada **5 minutos**.

---

## Antes vs agora

| Antes (stack clássico) | Agora (Go) |
|------------------------|------------|
| `mailgraph.pl` + `mailgraph.cgi` | Um binário `mailgraph` |
| Apache2 | [Echo v5](https://echo.labstack.com/) embutido |
| Gráficos PNG estáticos | Gráficos interativos no navegador |
| Imagem Debian + Perl | **Alpine ~31 MB** + binário UPX **~3,2 MB** |

Inclui o patch SPF / DMARC / DKIM de [Sebastian van de Meer](https://www.kernel-error.de/2014/04/22/mailgraph-graphen-um-spf-dmarc-und-dkim-erweitern/).

---

## Funcionalidades

- Leitura em tempo real do log (`tail -f`)
- Compatível com arquivos RRD do Mailgraph original
- Períodos com URL própria (compartilhe links como `/last-week`)
- HTTPS e HTTP Basic Auth opcionais
- Interface na raiz (`/`) — sem prefixo `/mailgraph`
- CSS embutido no binário (`go:embed` em `main.go`)
- Postfix, Sendmail, Exim, Amavis, ClamAV, SpamAssassin e outros

## Stack técnica

- **Go** 1.26 · **Cobra** + **Viper** · **Echo** v5 · **go-echarts** v2 · **rrdtool** · **UPX**

## Estrutura do projeto

```
main.go                 # entrypoint; go:embed de web/static/mailgraph.css
cmd/                    # CLI (server, cat, version, generate-config)
web/static/             # CSS e assets (embutidos no binário)
internal/               # collector, rrd, charts, web, config…
docs/screenshots/       # diagramas e imagens da documentação
Dockerfile              # multi-stage: Go + UPX → Alpine
docker-compose.test.yml # teste local na porta 8585
Makefile                # build, Docker, fetch de log remoto
```

---

## CLI

```bash
mailgraph server           # coletor + servidor HTTP (padrão no Docker)
mailgraph cat              # processa o log uma vez e sai
mailgraph version          # versão e build info
mailgraph generate-config  # gera config.toml a partir do template
```

No container, `entrypoint.sh` executa `mailgraph server` por padrão.

---

## Interface web (rotas)

| Rota | Descrição |
|------|-----------|
| `/today`, `/last-day`, … | Página com 6 gráficos do período |
| `/mailgraph.css` | CSS embutido no binário |
| `/chart?period=N&type=T` | HTML de um gráfico (`T` = `n`/`e`/`s`/`d`/`k`/`v`) |

No eixo horizontal: **data e hora** (`MM-DD HH:MM`, horário local do servidor).

---

## Configuração

Prioridade: **flags** > **variáveis `MAILGRAPH_*`** > **`config.toml`** > **padrões**.

Arquivos procurados: `./config.toml`, `/etc/mailgraph/config.toml`, `~/.mailgraph/config.toml`

### Exemplo mínimo (`config.toml`)

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

Gere um arquivo de partida com `mailgraph generate-config` ou copie `config.toml.example`.

<details>
<summary><strong>HTTPS, Basic Auth e variáveis de ambiente</strong> (clique para expandir)</summary>

### HTTP Basic Auth

```toml
[auth]
enabled = true
username = "admin"
password = "secret"
realm = "Mailgraph"
```

### HTTPS (TLS)

```bash
make certs   # ssl/server.crt e ssl/server.key (teste local)

mailgraph server --listen=:8443 --tls \
  --tls-cert=ssl/server.crt --tls-key=ssl/server.key
```

Gráficos: **https://localhost:8443/today**

### Variáveis de ambiente

| Variável | Equivalente |
|----------|-------------|
| `MAILGRAPH_LOG_FILE` | `log.file` |
| `MAILGRAPH_RRD_DIR` | `rrd.dir` |
| `MAILGRAPH_SERVER_LISTEN` | `server.listen` |
| `MAILGRAPH_SERVER_HOSTNAME` | `server.hostname` |
| `MAILGRAPH_SERVER_TLS_ENABLED` | `server.tls_enabled` |
| `MAILGRAPH_AUTH_ENABLED` | `auth.enabled` |
| `MAILGRAPH_AUTH_USERNAME` | `auth.username` |
| `MAILGRAPH_AUTH_PASSWORD` | `auth.password` |

### Flags principais

| Flag | Descrição |
|------|-----------|
| `--logfile` | Arquivo de log syslog |
| `--daemon-rrd` | Diretório dos `.rrd` |
| `--hostname` | Nome no título dos gráficos |
| `--listen` | Endereço de escuta (padrão `:8080`) |
| `--tls` / `--tls-cert` / `--tls-key` | HTTPS |
| `--auth` / `--auth-user` / `--auth-pass` | Basic Auth |
| `--ignore-localhost` | Ignora tráfego de/para `127.0.0.1` |

</details>

---

## Build

```bash
make deps          # módulos Go
make build         # bin/mailgraph (~11 MB)
make build-prod    # bin/mailgraph + UPX (~3,2 MB)
make build-docker  # imagem Docker
make test          # go test ./...
make help
```

Requisitos: Go 1.26+, `rrdtool` (runtime), `make`, UPX (opcional).

---

## Docker (opções avançadas)

### Com TLS

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

### Com Basic Auth

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

### Teste local com log remoto

```bash
make fetch-testdata TESTDATA_HOST=mx01    # testdata/mail.log (gitignored)
make test-docker                          # http://127.0.0.1:8585/today
make test-docker-down
```

Para reprocessar o log do zero: `rm -rf testdata/rrd/*` antes de subir o container.

---

## Como os dados são atualizados

| Etapa | Intervalo |
|-------|-----------|
| Leitura do log | Tempo real (`tail -f`) |
| Gravação no RRD | Buckets de **1 minuto** |
| Atualização da página | **5 minutos** (meta refresh) |

---

## Screenshots

### Last week

![Last week](screenshots/lastweek.png)

### Last month

![Last month](screenshots/lastmonth.png)

---

## Créditos

- [Mailgraph](https://mailgraph.schweikert.ch) — David Schweikert (GPL)
- Patch SPF/DMARC/DKIM — Sebastian van de Meer
- Container Docker original — [David Ullrich](https://www.production-ready.de/2023/04/15/mailgraph-docker-container-en.html) ([DE](https://www.production-ready.de/2023/04/15/mailgraph-docker-container.html))
- Port Go — MailgraphContainer

## Licença

GNU General Public License v2 — ver `backups/mailgraph/COPYING` (código original).