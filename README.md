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

## Stack

- **Go** 1.26
- **Echo** v5 — servidor HTTP
- **go-echarts** v2 — gráficos na web
- **rrdtool** — armazenamento de séries temporais (runtime)
- **UPX** — compressão do binário em builds de produção

## Estrutura do projeto

```
cmd/mailgraph/          # entrypoint
internal/
  collector/            # tail do log + parsing de eventos
  syslog/               # parser syslog/metalog
  rrd/                  # create/update/fetch via rrdtool
  charts/               # geração de gráficos go-echarts
  web/                  # handlers Echo v5
  config/               # flags CLI
Dockerfile              # build multi-stage (Go + UPX → Alpine)
Makefile                # build local, UPX, Docker
entrypoint.sh           # entrypoint do container
mailgraph/              # scripts Perl originais (referência)
```

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
./bin/mailgraph --version
```

### Flags principais

```bash
mailgraph \
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
| `--listen` | Endereço HTTP (padrão `:8080`) |
| `--ignore-localhost` | Ignora tráfego de/para `127.0.0.1` |
| `-c` / `--cat` | Processa o log uma vez e sai |
| `--help` | Ajuda completa |

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

Gráficos: **http://localhost:8080/mailgraph/**

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

### Reverse proxy com Traefik

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
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.mailgraph-router.rule=Host(`mail.example.com`) && PathPrefix(`/mailgraph`)"
      - "traefik.http.routers.mailgraph-router.entryPoints=websecure"
      - "traefik.http.routers.mailgraph-router.service=mailgraph-service"
      - "traefik.http.services.mailgraph-service.loadBalancer.server.scheme=http"
      - "traefik.http.services.mailgraph-service.loadBalancer.server.port=8080"
      - "traefik.http.routers.mailgraph-router.middlewares=mailgraph-middleware-auth"
      - "traefik.http.middlewares.mailgraph-middleware-auth.basicauth.users=user:[password-hash]"
```

---

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

GNU General Public License v2 — ver `mailgraph/COPYING` (código original).