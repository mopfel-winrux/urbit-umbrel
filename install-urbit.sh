#!/bin/bash
set -e
DEVICE_ARCH=$(uname -m)
if [[ $DEVICE_ARCH == "aarch64" ]]; then
  mkdir -p /urbit/binary
  cd /urbit/binary/
  wget https://github.com/mopfel-winrux/urbit-umbrel/releases/download/urbit-v1.9/urbit-v1.9-aarch64-linux.tgz
  tar zxvf ./urbit-v1.9-aarch64-linux.tgz --strip=1
  mv /urbit/binary/urbit* /usr/sbin/

elif [[ $DEVICE_ARCH == "x86_64" ]]; then
  mkdir -p /urbit/binary
  cd /urbit/binary/
  wget --content-disposition https://urbit.org/install/linux64/latest
  tar zxvf ./linux64.tgz --strip=1
  mv /urbit/binary/urbit* /usr/sbin/
fi
