#!/bin/bash

keyname=$1
AMES_PORT=$2
LOOM_VALUE=$3

echo "Urbit will use $keyname to create a pier and then restart. Please wait while it initializes your ship."

urbit -t -x -b 0.0.0.0 -p $AMES_PORT --loom $LOOM_VALUE -w $(basename $keyname .key) -k $keyname -c /data/piers/$(basename $keyname .key) 


# Remove the keyfile for security
echo "Urbit will delete the key for security reasons."
rm "$keyname"
