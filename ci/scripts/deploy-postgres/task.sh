#!/bin/bash -eu

preflight_check() {
  set +x
  test -n "${BOSH_DIRECTOR}"
  test -n "${BOSH_USER}"
  test -n "${BOSH_PASSWORD}"
  set -x
}

deploy() {
  bosh \
    -n \
    -t "${1}" \
    -d "${2}" \
    deploy
}

function upload_remote_release() {
  local release_url=$1
  wget --quiet "${release_url}" -O remote_release.tgz
  bosh upload release remote_release.tgz
}

upload_stemcell() {
  pushd /tmp > /dev/null
    curl -Ls -o /dev/null -w %{url_effective} https://bosh.io/d/stemcells/bosh-softlayer-xen-ubuntu-trusty-go_agent | xargs -n 1 curl -O
    bosh upload stemcell light-bosh-stemcell-*-softlayer-xen-ubuntu-trusty-go_agent.tgz --skip-if-exists
  popd > /dev/null
}

generate_env_stub() {
  cat <<EOF
---
common_data:
  <<: (( merge ))
  VmNamePrefix: pgci-postgres-
  env_name: pgci-postgres
  default_env:
    bosh:
      password: ~
EOF
}

function main(){
  local root="${1}"

  set +x
  bosh target https://${BOSH_DIRECTOR}:25555
  bosh login ${BOSH_USER} ${BOSH_PASSWORD}
  set -x
  mkdir stubs

  upload_remote_release "https://bosh.io/d/github.com/cloudfoundry/postgres-release"
  upload_stemcell
  pushd stubs
    generate_env_stub > env.yml
  popd

  pushd "${root}/postgres-release"
    spiff merge \
      "${root}/postgres-ci-env/deployments/postgres/pgci-postgres.yml" \
      "${root}/stubs/env.yml" \
      "${root}/postgres-ci-env/deployments/common/common.yml" > "${root}/pgci-postgres.yml"
  popd

  deploy \
    "${BOSH_DIRECTOR}" \
    "${root}/pgci-postgres.yml"

}


main "${PWD}"
