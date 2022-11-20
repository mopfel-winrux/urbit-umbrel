#!/bin/bash
set -e
DEVICE_ARCH=$(uname -m)
if [[ $DEVICE_ARCH == "aarch64" ]]; then
  mkdir -p /urbit/binary
  cd /urbit/binary/
  wget https://github.com/urbit/urbit/releases/download/urbit-v1.12/linux-aarch64.tgz
  tar zxvf ./linux-aarch64.tgz --strip=1
  mv /urbit/binary/urbit /usr/sbin/

elif [[ $DEVICE_ARCH == "x86_64" ]]; then
  mkdir -p /urbit/binary
  cd /urbit/binary/
  wget --content-disposition https://urbit.org/install/linux64/latest
  tar zxvf ./linux64.tgz --strip=1
  mv /urbit/binary/urbit* /usr/sbin/
fi
