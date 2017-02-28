#!/bin/bash -exu

function main() {
  export BOSH_ENVIRONMENT="https://${BOSH_DIRECTOR}:25555"
  bosh clean-up --all
}

main
