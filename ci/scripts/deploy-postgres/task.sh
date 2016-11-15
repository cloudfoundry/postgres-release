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

upload_remote_release() {
  local release_url=$1
  wget --quiet "${release_url}" -O remote_release.tgz
  bosh upload release remote_release.tgz
}

generate_dev_release_stub() {
  local build_dir
  build_dir="${1}"

  cat <<EOF
---
releases:
- name: postgres
  version: create
  url: file://${build_dir}/postgres-release
EOF
}

generate_uploaded_release_stub() {
  local release_version
  release_version="${1}"

  cat <<EOF
---
releases:
- name: postgres
  version: ${release_version}
EOF
}

upload_stemcell() {
  pushd /tmp > /dev/null
    curl -Ls -o /dev/null -w %{url_effective} https://bosh.io/d/stemcells/bosh-softlayer-xen-ubuntu-trusty-go_agent | xargs -n 1 curl -O
    bosh upload stemcell light-bosh-stemcell-*-softlayer-xen-ubuntu-trusty-go_agent.tgz --skip-if-exists
  popd > /dev/null
}

generate_env_stub() {
local vm_prefix
vm_prefix="${PG_DEPLOYMENT}-"
  cat <<EOF
---
common_data:
  <<: (( merge ))
  VmNamePrefix: ${vm_prefix}
  env_name: ${PG_DEPLOYMENT}
EOF
}

function main(){
  local root="${1}"

  set +x
  bosh target https://${BOSH_DIRECTOR}:25555
  bosh login ${BOSH_USER} ${BOSH_PASSWORD}
  set -x
  mkdir stubs

  upload_stemcell
  pushd stubs
    if [ "${PG_VERSION}" == "master" ]; then
      upload_remote_release "https://bosh.io/d/github.com/cloudfoundry/postgres-release"
      generate_uploaded_release_stub "latest" > releases.yml
    elif [ "${PG_VERSION}" == "develop" ]; then
      generate_dev_release_stub ${root} > releases.yml
    else
      upload_remote_release "https://bosh.io/d/github.com/cloudfoundry/postgres-release?v=${PG_VERSION}"
      generate_uploaded_release_stub ${PG_VERSION} > releases.yml
    fi
    generate_env_stub > env.yml
  popd

  pushd "${root}/postgres-release"
    spiff merge \
      "${root}/postgres-ci-env/deployments/postgres/iaas-infrastructure.yml" \
      "${root}/stubs/env.yml" \
      "${root}/postgres-ci-env/deployments/common/common.yml" > "${root}/iaas.yml"
    spiff merge \
      "${root}/postgres-ci-env/deployments/postgres/properties.yml" \
      "${root}/stubs/env.yml" \
      "${root}/postgres-ci-env/deployments/common/common.yml" > "${root}/props.yml"
    scripts/generate-deployment-manifest \
      -i "${root}/iaas.yml" \
      -p "${root}/props.yml" \
      -v "${root}/stubs/releases.yml" > "${root}/pgci-postgres.yml"
  popd

  deploy \
    "${BOSH_DIRECTOR}" \
    "${root}/pgci-postgres.yml"

}


main "${PWD}"
