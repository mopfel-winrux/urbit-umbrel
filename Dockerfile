# syntax=docker/dockerfile:1

FROM python:3.9-slim-buster

RUN apt-get update && apt-get --no-install-recommends install -y curl wget vim ca-certificates gnupg python3-pip procps nginx apt-utils
COPY install-urbit.sh /tmp/install-urbit.sh
RUN  chmod +x /tmp/install-urbit.sh && /tmp/install-urbit.sh && rm /tmp/install-urbit.sh

ENV FLASK_APP=app
ENV FLASK_RUN_HOST=0.0.0.0

COPY requirements.txt requirements.txt
RUN pip3 install -r requirements.txt

COPY ./app /tmp/app
COPY nginx.conf /etc/nginx/conf.d/nginx.conf
RUN mkdir /data/
ADD start.sh /usr/bin/start.sh
RUN chmod +x /usr/bin/start.sh
EXPOSE 8090

#CMD [ "python3", "-m" , "flask", "run", "--host=0.0.0.0"]
ENTRYPOINT ["/usr/bin/start.sh"]

