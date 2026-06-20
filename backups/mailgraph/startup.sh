#!/bin/sh

if [ "$#" -gt 0 ]; then
	exec "$@"
fi

echo "Starting mailgraph (Go).."

exec /usr/local/bin/mailgraph \
  --logfile=/var/log/mail/mail.log \
  --daemon-rrd=/var/www/mailgraph/rrd \
  --listen=:8080