# syntax=docker/dockerfile:1
# RPC builder

FROM python:3.9-slim-buster as ship-runner

ARG UID=1000
ARG GID=1000
ARG username=umbrel
RUN apt-get update && apt-get install -y curl
RUN curl -sL https://deb.nodesource.com/setup_14.x | bash -
RUN apt-get update && apt-get --no-install-recommends install -y curl wget vim ca-certificates gnupg python3-pip procps nginx apt-utils sudo nodejs git
COPY install-urbit.sh /tmp/install-urbit.sh
RUN  chmod +x /tmp/install-urbit.sh && /tmp/install-urbit.sh && rm /tmp/install-urbit.sh

ADD https://api.github.com/repos/urbit/urbit-bitcoin-rpc/git/refs/heads/master version.json
RUN git clone -b master https://github.com/urbit/urbit-bitcoin-rpc.git urbit-bitcoin-rpc

ENV FLASK_APP=app
ENV FLASK_RUN_HOST=0.0.0.0

COPY requirements.txt requirements.txt
RUN pip3 install -r requirements.txt

COPY ./server.js /src/
COPY ./app /tmp/app
RUN mv /urbit-bitcoin-rpc/* /
RUN npm install express
RUN npm audit fix
COPY nginx.conf /etc/nginx/conf.d/nginx.conf
RUN mkdir /data/
ADD start.sh /usr/bin/start.sh
RUN chmod +x /usr/bin/start.sh
USER $USERNAME
EXPOSE 8090
EXPOSE 50002

#CMD [ "python3", "-m" , "flask", "run", "--host=0.0.0.0"]
ENTRYPOINT ["/usr/bin/start.sh"]

