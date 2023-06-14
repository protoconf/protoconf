#!/bin/bash
set -e
set -x
echo "VARS: $*"
STORE_TYPE=${STORE_TYPE:-etcd}
STORE_ADDRESS=${STORE_ADDRESS:-localhost:2379}
PREFIX=${PREFIX:-"prod"}
BULK_SIZE=${BULK_SIZE:-10}
CONFIGS_TO_INSERT=$(find  materialized_config/ -type  f | sed -e 's/^materialized_config\/\(.*\)/\1/')
if [ -z "${CONFIGS_TO_INSERT}" ]; then
    exit 0
fi
echo ${CONFIGS_TO_INSERT} | xargs -n${BULK_SIZE} \
    time protoconf insert \
        -store "${STORE_TYPE}" \
        -store-address "${STORE_ADDRESS}" \
        -prefix "${PREFIX}" \
        .