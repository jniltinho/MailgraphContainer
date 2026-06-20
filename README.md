# Mailgraph (Go)

Port em **Golang** do [Mailgraph](https://mailgraph.schweikert.ch) — frontend de estatísticas de e-mail baseado em RRDtool para **Postfix** e outros MTAs.

Um único binário substitui o stack antigo (Perl + Apache + CGI + PNG estático):

| Antes | Agora |
|-------|-------|
| `mailgraph.pl` + `mailgraph.cgi` | `mailgraph` (Go) |
| Apache2 | [Echo v5](https://echo.labstack.com/) |
| Gráficos PNG (`rrdtool graph`) | [go-echarts](https://github.com/go-echarts/go-echarts) (interativos) |
| Imagem Debian + Perl | Alpine + binário UPX (~2.7 MB) |

Inclui o patch SPF / DMARC / DKIM de [Sebastian van de Meer](https://www.kernel-error.de/2014/04/22/mailgraph-graphen-um-spf-dmarc-und-dkim-erweitern/).

**Instalação em servidor Postfix:** [README-POSTFIX.md](README-POSTFIX.md)

Imagem Docker: [Docker Hub — davidullrich/mailgraph](https://hub.docker.com/r/davidullrich/mailgraph)

---

## Funcionalidades

- Leitura em tempo real do log de e-mail (`tail -f`)
- Compatível com arquivos RRD existentes do Mailgraph original
- Gráficos: enviados/recebidos, erros, SPF, DMARC, DKIM, Dovecot, vírus/spam
- Períodos: dia, semana, 2 semanas, mês, 2 meses, ano, 2 anos
- Suporte a Postfix, Sendmail, Exim, Amavis, ClamAV, SpamAssassin e outros
- HTTPS opcional via TLS nativo do Echo v5
- HTTP Basic Auth opcional (middleware do Echo)
- Interface web na raiz (`/`), sem prefixo `/mailgraph`

## Stack

- **Go** 1.26
- **Cobra** + **Viper** — CLI e configuração (`config.toml`, variáveis de ambiente)
- **Echo** v5 — servidor HTTP/HTTPS
- **go-echarts** v2 — gráficos na web
- **rrdtool** — armazenamento de séries temporais (runtime)
- **UPX** — compressão do binário em builds de produção

## Estrutura do projeto

```
main.go                 # entrypoint
cmd/                    # comandos Cobra (server, cat, version, generate-config)
internal/
  buildinfo/            # versão (ldflags)
  config/               # carregamento Viper
  collector/            # tail do log + parsing de eventos
  syslog/               # parser syslog/metalog
  rrd/                  # create/update/fetch via rrdtool
  charts/               # geração de gráficos go-echarts
  web/                  # handlers Echo v5
config.toml.example     # exemplo de configuração
Dockerfile              # build multi-stage (Go + UPX → Alpine)
Makefile                # build local, UPX, Docker
entrypoint.sh           # entrypoint do container (mailgraph server)
backups/mailgraph/      # scripts Perl originais (referência)
```

---

## CLI

```bash
mailgraph server           # coletor + servidor HTTP/HTTPS (padrão no Docker)
mailgraph cat              # processa o log uma vez e sai
mailgraph version          # versão e build info
mailgraph generate-config  # gera config.toml a partir do template embutido
mailgraph --help           # ajuda geral
mailgraph server --help    # flags do subcomando server
```

No container, `entrypoint.sh` executa `mailgraph server` por padrão. Argumentos passados ao `docker run` substituem esse comportamento.

---

## Configuração

Prioridade (maior → menor): **flags** > **variáveis `MAILGRAPH_*`** > **`config.toml`** > **padrões**.

Arquivos de configuração procurados automaticamente:

1. `./config.toml`
2. `/etc/mailgraph/config.toml`
3. `~/.mailgraph/config.toml`

Use `--config /caminho/config.toml` para um arquivo específico.

### Exemplo (`config.toml`)

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
tls_enabled = false
tls_cert = ""
tls_key = ""

[auth]
enabled = false
username = ""
password = ""
realm = "Mailgraph"

[filter]
ignore_localhost = true
```

### HTTP Basic Auth

```toml
[auth]
enabled = true
username = "admin"
password = "secret"
realm = "Mailgraph"
```

Ou via flags:

```bash
mailgraph server \
  --auth \
  --auth-user=admin \
  --auth-pass=secret
```

### HTTPS (TLS)

Para testes locais, gere certificado autoassinado:

```bash
make certs
# ssl/server.crt  ssl/server.key
```

```bash
mailgraph server \
  --listen=:8443 \
  --tls \
  --tls-cert=ssl/server.crt \
  --tls-key=ssl/server.key
```

Em produção, use certificado PEM (ex.: Let's Encrypt):

```toml
[server]
listen = ":8443"
tls_enabled = true
tls_cert = "/etc/ssl/certs/mailgraph.crt"
tls_key = "/etc/ssl/private/mailgraph.key"
```

Ou via flags:

```bash
mailgraph server \
  --listen=:8443 \
  --tls \
  --tls-cert=/etc/ssl/certs/mailgraph.crt \
  --tls-key=/etc/ssl/private/mailgraph.key
```

Gráficos em **https://localhost:8443/**

TLS + Basic Auth juntos:

```toml
[server]
listen = ":8443"
tls_enabled = true
tls_cert = "/etc/ssl/certs/mailgraph.crt"
tls_key = "/etc/ssl/private/mailgraph.key"

[auth]
enabled = true
username = "admin"
password = "secret"
```

Copie `config.toml.example` ou gere um arquivo com:

```bash
mailgraph generate-config
```

### Variáveis de ambiente

| Variável | Equivalente em `config.toml` |
|----------|------------------------------|
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

### Flags principais (`server` e `cat`)

```bash
mailgraph server \
  --logfile=/var/log/mail/mail.log \
  --daemon-rrd=/var/lib/mailgraph/rrd \
  --hostname=mail.example.com \
  --ignore-localhost \
  --listen=127.0.0.1:8080
```

| Flag | Descrição |
|------|-----------|
| `--logfile` | Arquivo de log syslog do Postfix |
| `--daemon-rrd` | Diretório dos arquivos `.rrd` |
| `--listen` | Endereço de escuta (padrão `:8080`) |
| `--hostname` | Nome exibido no título dos gráficos |
| `--tls` | Habilita HTTPS |
| `--tls-cert` | Arquivo do certificado TLS (PEM) |
| `--tls-key` | Arquivo da chave privada TLS (PEM) |
| `--auth` | Habilita HTTP Basic Auth |
| `--auth-user` | Usuário da autenticação |
| `--auth-pass` | Senha da autenticação |
| `--auth-realm` | Realm exibido no prompt do navegador |
| `--ignore-localhost` | Ignora tráfego de/para `127.0.0.1` |
| `--ignore-host` | Ignora host (regex, repetível) |
| `--verbose` | Saída detalhada |
| `--daemon` | Grava PID e desanexa do terminal |

---

## Build

### Requisitos

- Go 1.26+
- `rrdtool` (runtime)
- `make`
- UPX (opcional, para `build-prod` e Docker)

### Comandos

```bash
make deps          # baixar módulos Go
make build         # binário em bin/mailgraph
make build-prod    # build + UPX (--best --lzma)
make run           # build + mailgraph server (teste local)
make certs         # certificado TLS autoassinado em ssl/
make test          # go test ./...
make help          # lista completa
```

Build de produção com UPX (instalar uma vez):

```bash
make install-upx
make build-prod
```

Verificar versão:

```bash
./bin/mailgraph version
```

---

## Docker

Monte o log de e-mail e o diretório RRD:

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

Gráficos: **http://localhost:8080/**

Configuração opcional via arquivo ou ambiente:

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

Com TLS (monte certificado e chave):

```bash
docker run --rm -d \
  --name mailgraph \
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

Com Basic Auth:

```bash
docker run --rm -d \
  --name mailgraph \
  -v /var/log/mail/mail.log:/var/log/mail/mail.log:ro \
  -v /var/data/mailgraph/rrd:/var/www/mailgraph/rrd \
  -e MAILGRAPH_AUTH_ENABLED=true \
  -e MAILGRAPH_AUTH_USERNAME=admin \
  -e MAILGRAPH_AUTH_PASSWORD=secret \
  -p 8080:8080 \
  davidullrich/mailgraph:latest
```

Tag customizada:

```bash
make build-docker IMAGE=seu-usuario/mailgraph:latest
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

## Como funciona

| Etapa | Intervalo |
|-------|-----------|
| Leitura do log | Tempo real (`tail -f`) |
| Gravação no RRD | Buckets de **1 minuto** |
| Atualização da página web | **5 minutos** (meta refresh) |

Na primeira execução sem RRD, o histórico do log é importado automaticamente.

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