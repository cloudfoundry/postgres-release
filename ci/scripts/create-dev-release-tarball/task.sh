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
  bosh -t ${BOSH_DIRECTOR} create release --force --with-tarball --version "${REL_VERSION}" --name "${REL_NAME}"
  cp dev_releases/${REL_NAME}/${REL_NAME}-${REL_VERSION}.tgz ${root}/dev-release-tarball
  popd
}

main "${PWD}"
