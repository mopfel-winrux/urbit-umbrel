#!/bin/bash
set -e
DEVICE_ARCH=$(uname -m)
VERSION="v1.15"
mkdir -p /urbit/binary
cd /urbit/binary/
if [[ $DEVICE_ARCH == "aarch64" ]]; then
  curl -L https://github.com/urbit/urbit/releases/download/urbit-$VERSION/linux-aarch64.tgz | tar xzk --strip=1
elif [[ $DEVICE_ARCH == "x86_64" ]]; then
  curl -L https://github.com/urbit/urbit/releases/download/urbit-$VERSION/linux64.tgz | tar xzk --strip=1
fi
mv /urbit/binary/urbit /usr/sbin/
