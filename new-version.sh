# Script to build + push operator-sdk versions

#!/usr/bin/bash


set -e

VERSION=$1
CONTAINER="${2:-ricarhincapie/ns-operator}"

if [ -z ${VERSION} ]; then
  echo "You need to input a version"
  exit 1
else
  echo "Building version ${VERSION}..."
fi

echo "Building container"
make docker-build IMG=${CONTAINER}:${VERSION}
sleep 5


echo "Pushing container"
make docker-push IMG=${CONTAINER}:${VERSION}
sleep 5

echo "You're all set. Run:\n\
  make deploy IMG=${CONTAINER}:${VERSION}"

