#!/bin/bash
dirname=$1
AMES_PORT=$2

echo "Urbit detected a Pier named $dirname"
urbit -t -p $AMES_PORT piers/$(basename $dirname)
