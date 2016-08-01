#!/bin/bash -exu

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

generate_releases_stub() {
  local build_dir
  build_dir="${1}"

  cat <<EOF
---
releases:
- name: cf
  version: create
  url: file://${build_dir}/cf-release
- name: postgres
  version: create
  url: file://${build_dir}/postgres-release
EOF
}

generate_stemcell_stub() {
  pushd /tmp > /dev/null
    curl -Ls -o /dev/null -w %{url_effective} https://bosh.io/d/stemcells/bosh-softlayer-esxi-ubuntu-trusty-go_agent | xargs -n 1 curl -O
  popd > /dev/null

  local stemcell_filename
  stemcell_filename=$(echo /tmp/light-bosh-stemcell-*-softlayer-esxi-ubuntu-trusty-go_agent.tgz)

  local stemcell_version
  stemcell_version=$(echo ${stemcell_filename} | cut -d "-" -f4)

  cat <<EOF
---
meta:
  stemcell:
    name: bosh-softlayer-esxi-ubuntu-trusty-go_agent
    version: ${stemcell_version}
    url: file://${stemcell_filename}
EOF
}

generate_job_templates_stub() {
  cat <<EOF
meta:
  <<: (( merge ))
  postgres_templates:
  - name: postgres
    release: postgres
EOF
}

function main(){
  local root="${1}"

  mkdir stubs

  pushd stubs
    generate_releases_stub ${root} > releases.yml
    generate_stemcell_stub > stemcells.yml
    generate_job_templates_stub > job_templates.yml
  popd

  pushd "${root}/cf-release"
    spiff merge \
      "${root}/postgres-ci-env/deployments/cf/pgci-cf.yml" \
      "${root}/postgres-ci-env/deployments/common/properties.yml" \
      "${root}/postgres-ci-env/deployments/common/common.yml" > "${root}/partial-pgci-cf.yml"

    spiff merge \
      "templates/generic-manifest-mask.yml" \
      "templates/cf.yml" \
      "${root}/postgres-ci-env/deployments/cf/cf-infrastructure-softlayer.yml" \
      "${root}/stubs/releases.yml" \
      "${root}/stubs/stemcells.yml" \
      "${root}/stubs/job_templates.yml" \
      "${root}/partial-pgci-cf.yml" > "${root}/pgci_cf.yml"
  popd

  deploy \
    "${BOSH_DIRECTOR}" \
    "${root}/pgci_cf.yml"

  bosh -t $BOSH_DIRECTOR download manifest $HAPROXY_DEPLOYMENT ha_manifest.yml

  bosh -t ${BOSH_DIRECTOR} -d ha_manifest.yml -n restart ha_proxy 0
}


main "${PWD}"
