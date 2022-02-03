#!/bin/sh
mkdir -p /app/piers
mkdir -p /app/keys
nginx -c /etc/nginx/conf.d/nginx.conf
python3 -m flask run --host=0.0.0.0
