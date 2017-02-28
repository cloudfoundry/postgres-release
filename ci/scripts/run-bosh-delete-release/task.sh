#!/bin/bash -eu

function main() {
  export BOSH_ENVIRONMENT="https://${BOSH_DIRECTOR}:25555"
  bosh -n delete-release $REL_NAME/$REL_VERSION
}

main
