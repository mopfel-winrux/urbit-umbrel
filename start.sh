#!/bin/sh
cp -r /tmp/app/ /app

nginx -c /etc/nginx/conf.d/nginx.conf
cd /app
ls
env FLASK_APP=app.py python3 -m flask run --host=0.0.0.0
