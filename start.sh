#!/bin/sh
export BITCOIN_RPC_PORT=$BITCOIN_RPC_PORT
export BITCOIN_IP=$BITCOIN_IP
export ELECTRUM_IP=$ELECTRUM_IP
export ELECTRUM_PORT=$ELECTRUM_PORT
export BITCOIN_RPC_AUTH=$BITCOIN_RPC_AUTH
export BITCOIN_RPC_PASS=$BITCOIN_RPC_PASS
export APP_PASSWORD=$APP_PASSWORD
export PROXY_PORT=50002
cp -r /tmp/app/ /app
mkdir -p /etc/apache2
htpasswd -b -c /etc/apache2/.htpasswd umbrel $APP_PASSWORD
nginx -c /etc/nginx/conf.d/nginx.conf
cd /app
env FLASK_APP=app.py python3 -m flask run --host=0.0.0.0 &
