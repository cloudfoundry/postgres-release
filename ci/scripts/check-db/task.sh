#!/bin/bash -exu
preflight_check() {
  set +x
  test -n "${BOSH_DIRECTOR_IP}"
  test -n "${CF_DEPLOYMENT}"
  test -n "${API_USER}"
  test -n "${API_PASSWORD}"
  set -x
}

function main() {

  local root="${1}"
  preflight_check
  local api_endpoint="api.${BOSH_DIRECTOR_IP}.nip.io"

  cf api ${api_endpoint} --skip-ssl-validation
  set +x
  cf auth $API_USER $API_PASSWORD
  set -x
  cf target -o ${CF_DEPLOYMENT} -s ${CF_DEPLOYMENT}

  cf apps
  curl --fail dora.${BOSH_DIRECTOR_IP}.nip.io

}

main "${PWD}"
