#!/bin/bash -exu

preflight_check() {
  set +x
  test -n "${BOSH_DIRECTOR_IP}"
  test -n "${BOSH_DIRECTOR_NAME}"
  test -n "${BOSH_CLIENT}"
  test -n "${BOSH_CLIENT_SECRET}"
  test -n "${BOSH_CA_CERT}"
  set -x
}

function main() {
  preflight_check
  source ${root}/postgres-release/ci/scripts/configure_for_bosh.sh

  bosh -n clean-up --all
}

main
