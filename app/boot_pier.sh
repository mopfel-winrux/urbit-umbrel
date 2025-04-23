#!/bin/bash
dirname=$1
AMES_PORT=$2
LOOM_VALUE=$3

echo "Urbit detected a Pier named $dirname"
urbit -t -b 0.0.0.0 -p $AMES_PORT --loom $LOOM_VALUE /data/piers/$(basename $dirname)
