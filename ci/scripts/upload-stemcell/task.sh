#!/bin/bash -eu

preflight_check() {
  set +x
  test -n "${BOSH_DIRECTOR}"
  test -n "${BOSH_CLIENT}"
  test -n "${BOSH_CLIENT_SECRET}"
  set -x
}

function upload_stemcell() {
  wget --quiet 'https://bosh.io/d/stemcells/bosh-softlayer-xen-ubuntu-trusty-go_agent' --output-document=stemcell.tgz
  bosh upload-stemcell stemcell.tgz
}

function main(){
  local root="${1}"

  export BOSH_ENVIRONMENT="https://${BOSH_DIRECTOR}:25555"

  upload_stemcell
}

main "${PWD}"
