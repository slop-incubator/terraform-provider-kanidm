#!/bin/sh

set -e

SCRIPT_DIR="$(dirname "$(readlink -f "$0")")"

CONFIG_FILE=${CONFIG_FILE:="${SCRIPT_DIR}/insecure_server.toml"}

if [ -z "$KANIDM_TAG" ]; then
    KANIDM_TAG=latest
fi


# also where the files are stored
if [ -z "$KANI_TMP" ]; then
    KANI_TMP=/tmp/kanidm/
fi

if [ ! -d "${KANI_TMP}" ]; then
    echo "Creating temp kanidm dir: ${KANI_TMP}"
    mkdir -p "${KANI_TMP}"
else
    echo "Cleaning temp kanidm dir: ${KANI_TMP}"
    rm -rf "${KANI_TMP}"
fi

mkdir -p "${KANI_TMP}"/client_ca



if [ ! -f "${CONFIG_FILE}" ]; then
    echo "Couldn't find configuration file at ${CONFIG_FILE}, please ensure you're running this script from its base directory (${SCRIPT_DIR})."
    exit 1
fi

docker pull docker.io/kanidm/server:${KANIDM_TAG}
docker rm kanidev -f 2> /dev/null || true
docker create --name kanidev \
  -p '8443:8443' \
  -p '3636:3636' \
  -v $KANI_TMP:/data \
  docker.io/kanidm/server:${KANIDM_TAG}

docker cp $CONFIG_FILE kanidev:/data/server.toml
docker run --rm -v $KANI_TMP:/data \
  docker.io/kanidm/server:${KANIDM_TAG} \
  kanidmd cert-generate
docker start kanidev
