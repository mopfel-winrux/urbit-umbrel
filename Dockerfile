# syntax=docker/dockerfile:1

FROM debian:latest
#FROM python:3.8-slim-buster
RUN apt-get update && apt-get --no-install-recommends install -y curl wget vim ca-certificates gnupg python3-pip 

COPY install-urbit.sh /tmp/install-urbit.sh
RUN  chmod +x /tmp/install-urbit.sh && /tmp/install-urbit.sh && rm /tmp/install-urbit.sh

WORKDIR /app
ENV FLASK_APP=app.py
ENV FLASK_RUN_HOST=0.0.0.0

COPY requirements.txt requirements.txt
RUN pip3 install -r requirements.txt

COPY ./app /app

EXPOSE 5000

#CMD ["flask","run"]

#CMD ["/bin/bash"]
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
