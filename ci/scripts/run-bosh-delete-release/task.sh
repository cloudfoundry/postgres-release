#!/bin/bash -eu

function main() {
  set +x
  bosh target https://${BOSH_DIRECTOR}:25555
  bosh login ${BOSH_USER} ${BOSH_PASSWORD}
  set -x
  bosh -n --color delete release $REL_NAME $REL_VERSION
}

main
