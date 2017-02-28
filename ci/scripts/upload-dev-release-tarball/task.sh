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

  pushd ${root}/dev-release-tarball
  for file in *.tgz
  do
    echo loading release "$file"
    bosh upload-release "$file"
  done
  popd
}

main "${PWD}"
