#!/bin/bash
set -e
DEVICE_ARCH=$(uname -m)
VERSION="v1.16"
mkdir -p /urbit/binary
cd /urbit/binary/
if [[ $DEVICE_ARCH == "aarch64" ]]; then
  curl -L https://github.com/urbit/vere/releases/download/vere-$VERSION/linux-aarch64.tgz | tar xzk --strip=1
elif [[ $DEVICE_ARCH == "x86_64" ]]; then
  curl -L https://github.com/urbit/vere/releases/download/vere-$VERSION/linux-x86_64.tgz | tar xzk --strip=1
fi
mv /urbit/binary/urbit /usr/sbin/
