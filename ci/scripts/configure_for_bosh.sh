#!/bin/bash

set +x
test -n "${BOSH_DIRECTOR_IP}"
test -n "${BOSH_DIRECTOR_NAME}"
set -x

echo "${BOSH_DIRECTOR_IP} ${BOSH_DIRECTOR_NAME}" >> /etc/hosts
export BOSH_ENVIRONMENT="https://${BOSH_DIRECTOR_NAME}:25555"
