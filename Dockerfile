# syntax=docker/dockerfile:1

FROM debian:buster-slim

#FROM python:3.8-slim-buster

RUN apt-get update && apt-get --no-install-recommends install -y curl wget vim ca-certificates gnupg python3-pip procps nginx
COPY install-urbit.sh /tmp/install-urbit.sh
RUN  chmod +x /tmp/install-urbit.sh && /tmp/install-urbit.sh && rm /tmp/install-urbit.sh

WORKDIR /app
ENV FLASK_APP=app.py
ENV FLASK_RUN_HOST=0.0.0.0

COPY requirements.txt requirements.txt
RUN pip3 install -r requirements.txt

COPY ./app /app

RUN unlink /etc/nginx/sites-enabled/default
COPY reverse_proxy.conf /etc/nginx/sites-available/reverse_proxy.conf
RUN ln -s /etc/nginx/sites-available/reverse_proxy.conf /etc/nginx/sites-enabled/reverse_proxy.conf
RUN nginx -t
RUN service nginx restart


CMD [ "python3", "-m" , "flask", "run", "--host=0.0.0.0"]



#COPY start-urbit.sh /usr/sbin/start-urbit.sh
#RUN chmod +x /usr/sbin/start-urbit.sh && chmod +x /usr/sbin/get-urbit-code.sh && chmod +x /usr/sbin/reset-urbit-code.sh && chmod +x /usr/sbin/run-urbit-cmd.sh
#WORKDIR /urbit
#RUN mkdir piers && mkdir keys
#COPY app/keys/ keys/
#COPY app/piers/ piers/
#COPY install-urbit.sh /tmp/install-urbit.sh
#RUN echo "The Urbit's directory is populated with the following data:" && ls
#RUN  chmod +x /tmp/install-urbit.sh && /tmp/install-urbit.sh && rm /tmp/install-urbit.sh
#ENTRYPOINT ["/usr/sbin/start-urbit.sh"]
# ENTRYPOINT ["tail", "-f", "/dev/null"]
