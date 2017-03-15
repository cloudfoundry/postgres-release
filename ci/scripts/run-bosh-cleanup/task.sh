#!/bin/bash -exu

function main() {
  export BOSH_ENVIRONMENT="https://${BOSH_DIRECTOR}:25555"
  bosh -n clean-up --all
}

main
