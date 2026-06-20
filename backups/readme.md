
This project allows you to run Mailgraph inside a docker container.
It is based on the [Mailgraph](https://mailgraph.schweikert.ch) project from David Schweikert with dkim-, dmarc, spf-patch from [Sebastian van de Meer](https://www.kernel-error.de/2014/04/22/mailgraph-graphen-um-spf-dmarc-und-dkim-erweitern/). 

The docker image is available at [Docker Hub](https://hub.docker.com/r/davidullrich/mailgraph).

For more informations, see [this blog post](https://www.production-ready.de/2023/04/15/mailgraph-docker-container-en.html)
(german verison [here](https://www.production-ready.de/2023/04/15/mailgraph-docker-container.html)).

**Instalação em servidor Postfix (nativo ou Docker no host):** veja [README-POSTFIX.md](README-POSTFIX.md).

# Usage

To run the container you need to mount the mail.log file into the container to `/var/log/mail/mail.log` and provide a path or volume to store the rrd files at `/var/www/mailgraph/rrd`.

```bash
# Build local
make build-docker

# Run
docker run --rm -d \
  -v /var/log/mail/mail.log:/var/log/mail/mail.log \
  -v /var/data/mailgraph/rrd/:/var/www/mailgraph/rrd/ \
  -p 8080:8080 \
  davidullrich/mailgraph:latest
```

Graphs are served at http://localhost:8080/mailgraph/

This image runs a single Go binary (Echo v5 + go-echarts) instead of Apache/Perl.


## Docker Compose

``` yaml
version: '3'

services:
  mailgraph:
    image: davidullrich/mailgraph:latest
    hostname: mail.example.com
    volumes:
      - /var/log/mail/mail.log:/var/log/mail/mail.log
      - /var/data/mailgraph/rrd/:/var/www/mailgraph/rrd/
      - /etc/localtime:/etc/localtime:ro
    restart: unless-stopped
```

## Reverse Proxy with Traefik

To hide mailgraph behind a reverse proxy and add basic authentication with traefik you can use the following labels:

``` yaml
version: '3'

services:
  mailgraph:
    image: davidullrich/mailgraph:latest
    hostname: mail.example.com
    volumes:
      - /var/log/mail/mail.log:/var/log/mail/mail.log
      - /var/data/mailgraph/rrd/:/var/www/mailgraph/rrd/
      - /etc/localtime:/etc/localtime:ro
    restart: unless-stopped
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


# Screenshots

## Last week

![Last week](screenshots/lastweek.png)


## Last month

![Last month](screenshots/lastmonth.png)
