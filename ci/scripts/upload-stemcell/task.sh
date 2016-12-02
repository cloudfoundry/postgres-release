#!/bin/bash -eu

preflight_check() {
  set +x
  test -n "${BOSH_DIRECTOR}"
  test -n "${BOSH_USER}"
  test -n "${BOSH_PASSWORD}"
  set -x
}

function upload_stemcell() {
  wget --quiet 'https://bosh.io/d/stemcells/bosh-softlayer-xen-ubuntu-trusty-go_agent' --output-document=stemcell.tgz
  bosh upload stemcell stemcell.tgz --skip-if-exists
}

function main(){
  local root="${1}"

  set +x
  bosh target https://${BOSH_DIRECTOR}:25555
  bosh login ${BOSH_USER} ${BOSH_PASSWORD}
  set -x

  upload_stemcell
}

main "${PWD}"
