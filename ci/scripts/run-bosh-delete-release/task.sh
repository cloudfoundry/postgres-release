#!/bin/bash -eu

preflight_check() {
  set +x
  test -n "${BOSH_DIRECTOR_IP}"
  test -n "${BOSH_DIRECTOR_NAME}"
  test -n "${BOSH_CLIENT}"
  test -n "${BOSH_CLIENT_SECRET}"
  test -n "${BOSH_CA_CERT}"
  test -n "${REL_NAME}"
  test -n "${REL_VERSION}"
  set -x
}

function main() {
  local root="${1}"
  preflight_check
  source ${root}/postgres-release/ci/scripts/configure_for_bosh.sh

  bosh -n delete-release $REL_NAME/$REL_VERSION
}

main "${PWD}"
