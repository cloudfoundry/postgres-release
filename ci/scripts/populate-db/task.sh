#!/bin/bash -exu

preflight_check() {
  set +x
  test -n "${BOSH_DIRECTOR_IP}"
  test -n "${API_USER}"
  test -n "${API_PASSWORD}"
  test -n "${CF_DEPLOYMENT}"
  set -x
}

function main() {
  preflight_check

  local root="${1}"
  local api_endpoint="api.${BOSH_DIRECTOR_IP}.nip.io"

  cf api ${api_endpoint} --skip-ssl-validation
  set +x
  cf auth $API_USER $API_PASSWORD
  set -x

  cf create-org ${CF_DEPLOYMENT}
  cf target -o ${CF_DEPLOYMENT}
  cf create-space ${CF_DEPLOYMENT}
  cf target -s ${CF_DEPLOYMENT}

  pushd "${root}/cf-acceptance-tests/assets/dora"
    cf push dora
    cf apps
    curl --fail dora.${BOSH_DIRECTOR_IP}.nip.io
  popd

}

main "${PWD}"
