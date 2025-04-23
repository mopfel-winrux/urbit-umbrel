#!/bin/bash
AMES_PORT=$1
LOOM_VALUE=$2

random=$RANDOM
echo "Urbit did not detect any user selection. Booting a comet with the random name: comet-$random."
urbit -t -p $AMES_PORT --loom $LOOM_VALUE -c /data/piers/comet-$random
