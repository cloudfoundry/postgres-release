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

  pushd ${root}/dev-release
  git submodule update --init --recursive
  bosh -t ${BOSH_DIRECTOR} create release --force --version "${REL_VERSION}" --name "${REL_NAME}"
  bosh -t ${BOSH_DIRECTOR} upload release --version "${REL_VERSION}" --name "${REL_NAME}"
  popd
}

main "${PWD}"
