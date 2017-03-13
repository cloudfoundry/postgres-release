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
  /opt/rubies/ruby-2.2.4/bin/bosh target ${BOSH_ENVIRONMENT}
  /opt/rubies/ruby-2.2.4/bin/bosh login ${BOSH_CLIENT} ${BOSH_CLIENT_SECRET}

  pushd ${root}/dev-release
  /opt/rubies/ruby-2.2.4/bin/bosh -t ${BOSH_DIRECTOR} create release --force --with-tarball --version "${REL_VERSION}" --name "${REL_NAME}"
  cp dev_releases/${REL_NAME}/${REL_NAME}-${REL_VERSION}.tgz ${root}/dev-release-tarball

  #bosh create-release --force --tarball="${root}/dev-release-tarball/${REL_NAME}_${REL_VERSION}.tgz" --version "${REL_VERSION}" --name "${REL_NAME}"
  popd
}

main "${PWD}"
