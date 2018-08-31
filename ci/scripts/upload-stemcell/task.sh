#!/bin/bash -eu

preflight_check() {
  set +x
  test -n "${BOSH_DIRECTOR_IP}"
  test -n "${BOSH_DIRECTOR_NAME}"
  test -n "${BOSH_CLIENT}"
  test -n "${BOSH_CLIENT_SECRET}"
  test -n "${BOSH_CA_CERT}"
  test -n "${STEMCELL_VERSION}"
  test -n "${STEMCELL_TYPE}"
  set -x
}

function upload_stemcell() {
  if [ "${STEMCELL_VERSION}" == "latest" ]; then
    wget --quiet "https://bosh.io/d/stemcells/${STEMCELL_TYPE}-go_agent" --output-document=stemcell.tgz
  else
    wget --quiet "https://bosh.io/d/stemcells/${STEMCELL_TYPE}-go_agent?v=${STEMCELL_VERSION}" --output-document=stemcell.tgz
  fi
  bosh upload-stemcell stemcell.tgz
}

function main(){
  local root="${1}"
  preflight_check
  source ${root}/postgres-release/ci/scripts/configure_for_bosh.sh

  upload_stemcell
}

main "${PWD}"
