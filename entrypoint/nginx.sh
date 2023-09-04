#! /bin/ash

# install stuff
apk update
apk add openssl nginx bash curl

cd /usr/bin
curl -k https://i.jpillora.com/webproc | bash

# run nginx in foreground
webproc -c /etc/nginx/nginx.conf -c /etc/nginx/conf.d/proxy.conf -- nginx
