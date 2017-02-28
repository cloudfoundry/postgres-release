#!/bin/bash -eu

preflight_check() {
  set +x
  test -n "${BOSH_DIRECTOR}"
  test -n "${BOSH_CLIENT}"
  test -n "${BOSH_CLIENT_SECRET}"
  set -x
}

function main(){
  local root="${1}"

  export BOSH_ENVIRONMENT="https://${BOSH_DIRECTOR}:25555"

  pushd ${root}/dev-release
  bosh create-release --force --tarball="${root}/dev-release-tarball/${REL_NAME}_${REL_VERSION}.tgz" --version "${REL_VERSION}" --name "${REL_NAME}"
  popd
}

main "${PWD}"
