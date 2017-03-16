#!/bin/bash -exu

function main() {
  export BOSH_ENVIRONMENT="https://${BOSH_DIRECTOR}:25555"
  bosh -n -d $DEPLOYMENT_NAME run-errand acceptance_tests
}

main
