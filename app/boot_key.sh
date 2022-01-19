#!/bin/bash

keyname=$1
AMES_PORT=$2

echo "Urbit will use $keyname to create a pier and then restart. Please wait while it initializes your ship."

urbit -t -x -p $AMES_PORT -w $(basename $keyname .key) -k $keyname -c piers/$(basename $keyname .key) 


# Remove the keyfile for security
echo "Urbit will delete the key for security reasons."
rm "$keyname"
