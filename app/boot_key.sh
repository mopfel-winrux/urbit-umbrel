#!/bin/bash

keyname=$1
AMES_PORT=$2

echo "Urbit will use $keyname to create a pier and then restart. Please wait while it initializes your ship."

urbit -x -p $AMES_PORT -w $(basename $keyname .key) -k $keyname -c piers/$(basename $keyname .key) || echo "Urbit can't initialize a pier with the key $keyname" \
&& echo "It's possible that there is already an initialized pier with that keyfile." && echo "Urbit will now delete the keyfile and reboot."

# Remove the keyfile for security
echo "Urbit will delete the key for security reasons."
rm "$keyname"
echo "Urbit will now restart the container."
