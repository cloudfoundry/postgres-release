#!/bin/bash -eu

preflight_check() {
  set +x
  test -n "${BOSH_DIRECTOR}"
  test -n "${BOSH_USER}"
  test -n "${BOSH_PASSWORD}"
  set -x
}

function main(){
  local root="${1}"

  set +x
  bosh target https://${BOSH_DIRECTOR}:25555
  bosh login ${BOSH_USER} ${BOSH_PASSWORD}
  set -x

  pushd ${root}/dev-release-tarball
  for file in *.tgz
  do
    echo loading release "$file"
    bosh -t ${BOSH_DIRECTOR} upload release "$file"
  done
  popd
}

main "${PWD}"
