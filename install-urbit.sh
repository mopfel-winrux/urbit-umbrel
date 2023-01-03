#!/bin/bash
set -e
DEVICE_ARCH=$(uname -m)
mkdir -p /urbit/binary
cd /urbit/binary/
if [[ $DEVICE_ARCH == "aarch64" ]]; then
  curl -L https://urbit.org/install/linux-aarch64/latest | tar xzk --strip=1
elif [[ $DEVICE_ARCH == "x86_64" ]]; then
  curl -L https://urbit.org/install/linux64/latest | tar xzk --strip=1
fi
mv /urbit/binary/urbit /usr/sbin/
