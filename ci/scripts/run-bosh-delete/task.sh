#!/bin/bash -eu

preflight_check() {
  set +x
  test -n "${BOSH_DIRECTOR_IP}"
  test -n "${BOSH_DIRECTOR_NAME}"
  test -n "${BOSH_CLIENT}"
  test -n "${BOSH_CLIENT_SECRET}"
  test -n "${BOSH_CA_CERT}"
  test -n "${DEPLOYMENT_NAME}"
  set -x
}

function main() {
  preflight_check
  source postgres-release/ci/configure_for_bosh.sh

  bosh -n -d $DEPLOYMENT_NAME delete-deployment --force
}

main
