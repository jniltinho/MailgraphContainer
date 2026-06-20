# Stage 1: Build the Go application
FROM golang:1.26-alpine AS go-builder

ARG VERSION=dev
ARG GIT_COMMIT=unknown

ENV TZ=America/Sao_Paulo

RUN apk add --no-cache upx tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) && \
    CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w \
    -X mailgraph/cmd.Version=${VERSION} \
    -X mailgraph/cmd.BuildDate=${BUILD_DATE} \
    -X mailgraph/cmd.GitCommit=${GIT_COMMIT}" \
    -o bin/mailgraph . && \
    upx --best --lzma bin/mailgraph

# Stage 2: Final runtime image
FROM alpine:3.21

ENV TZ=America/Sao_Paulo

RUN apk add --no-cache \
        ca-certificates \
        tzdata \
        rrdtool

WORKDIR /app

COPY --from=go-builder /app/bin/mailgraph /usr/local/bin/mailgraph
COPY entrypoint.sh /app/entrypoint.sh
COPY config.toml.example /etc/mailgraph/config.toml.example

RUN chmod +x /app/entrypoint.sh && \
    mkdir -p /var/www/mailgraph/rrd /etc/mailgraph

VOLUME ["/var/log/mail/mail.log", "/var/www/mailgraph/rrd"]

EXPOSE 8080

ENTRYPOINT ["/app/entrypoint.sh"]