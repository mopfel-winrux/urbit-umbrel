#!/bin/bash
set -e
DEVICE_ARCH=$(uname -m)
if [[ $DEVICE_ARCH == "aarch64" ]]; then
  #curl https://s3.us-east-2.amazonaws.com/urbit-on-arm/urbit-on-arm_public.gpg | apt-key add -
  #echo 'deb http://urbit-on-arm.s3-website.us-east-2.amazonaws.com buster custom' | tee /etc/apt/sources.list.d/urbit-on-arm.list
  #dpkg --add-architecture arm64
  #apt-get update
  #apt-get install -y urbit:arm64
  #rm -rf /var/lib/apt/lists/*
  mkdir -p /urbit/binary
  cd /urbit/binary/
  wget https://github.com/botter-nidnul/urbit/releases/download/urbit-v1.8-aarch64/urbit-v1.8-aarch64-linux.tgz
  tar zxvf ./urbit-v1.8-aarch64-linux.tgz --strip=1
  mv /urbit/binary/urbit* /usr/sbin/

elif [[ $DEVICE_ARCH == "x86_64" ]]; then
  mkdir -p /urbit/binary
  cd /urbit/binary/
  wget --content-disposition https://urbit.org/install/linux64/latest
  tar zxvf ./linux64.tgz --strip=1
  mv /urbit/binary/urbit* /usr/sbin/
fi
