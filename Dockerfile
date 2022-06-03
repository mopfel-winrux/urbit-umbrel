# syntax=docker/dockerfile:1
# RPC builder

FROM python:3.9-slim-buster as ship-runner

ARG UID=1000
ARG GID=1000
ARG username=umbrel
RUN apt-get update && apt-get --no-install-recommends install -y curl wget vim ca-certificates gnupg python3-pip procps nginx apt-utils sudo git apache2-utils
COPY install-urbit.sh /tmp/install-urbit.sh
RUN  chmod +x /tmp/install-urbit.sh

ENV FLASK_APP=app
ENV FLASK_RUN_HOST=0.0.0.0

COPY requirements.txt requirements.txt
RUN pip3 install -r requirements.txt

COPY ./app /tmp/app
COPY nginx.conf /etc/nginx/conf.d/nginx.conf
RUN mkdir /data/
ADD start.sh /usr/bin/start.sh
RUN chmod +x /usr/bin/start.sh
USER $USERNAME
EXPOSE 8090

ENTRYPOINT ["/usr/bin/start.sh"]

