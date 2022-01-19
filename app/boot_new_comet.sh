keyname=$1
AMES_PORT=$2

random=$RANDOM
echo "Urbit did not detect any user selection. Booting a comet with the random name: comet-$random."
urbit -t -p $AMES_PORT -c piers/comet-$random
