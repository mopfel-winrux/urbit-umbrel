#!/bin/bash
dirname=$1
AMES_PORT=$2

echo "Urbit detected a Pier named $dirname"
urbit -t -b 0.0.0.0 -p $AMES_PORT /data/piers/$(basename $dirname)
