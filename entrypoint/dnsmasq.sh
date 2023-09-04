#! /bin/ash

apk update
apk add dnsmasq curl bash

cd /usr/bin/
curl https://i.jpillora.com/webproc | bash

webproc -c /etc/dnsmasq.conf -- dnsmasq --no-daemon -q
