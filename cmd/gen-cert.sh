#!/bin/bash
SSLCOMMAND=$(which openssl)

if [[ ! -d "~/.config/tiamat" ]]; then
  mkdir ~/.config/tiamat
fi

$SSLCOMMAND genrsa -out ~/.config/tiamat/server.key 2048

$SSLCOMMAND req -new -x509 -sha256 -key ~/.config/tiamat/server.key -out ~/.config/tiamat/server.crt -days 3650 -subj "/C=ES/ST=Valencia/L=Valencia/O=mini-chat /OU=tiamat/CN=tiamat"
