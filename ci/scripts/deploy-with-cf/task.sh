#!/bin/bash -eu

preflight_check() {
  set +x
  test -n "${BOSH_DIRECTOR}"
  test -n "${BOSH_CLIENT}"
  test -n "${BOSH_CLIENT_SECRET}"
  set -x
}

generate_releases_stub() {
  local build_dir
  build_dir="${1}"

  cat <<EOF
---
releases:
- name: cf
  version: ${REL_VERSION}
- name: postgres
  version: ${REL_VERSION}
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

generate_env_stub() {
  local vm_prefix
  local cf1_domain
  local apps_domain
  local haproxy_instances
  vm_prefix="${CF_DEPLOYMENT}-"
  apps_domain="apps.${CF_DEPLOYMENT}.microbosh"
  cf1_domain="cf1.${CF_DEPLOYMENT}.microbosh"
  haproxy_instances=1
  cat <<EOF
---
common_data:
  <<: (( merge ))
  VmNamePrefix: ${vm_prefix}
  cf1_domain: ${cf1_domain}
  env_name: ${CF_DEPLOYMENT}
  apps_domain: ${apps_domain}
  api_user: ${API_USER}
  api_password: ${API_PASSWORD}
  haproxy_instances: ${haproxy_instances}
  Bosh_ip: ${BOSH_DIRECTOR}
  Bosh_public_ip: ${BOSH_PUBLIC_IP}
  stemcell_version: ${STEMCELL_VERSION}
  default_env:
    bosh:
      password: ~
      keep_root_password: true
EOF
}

function main(){
  local root="${1}"

  export BOSH_ENVIRONMENT="https://${BOSH_DIRECTOR}:25555"
  mkdir stubs

  pushd stubs
    generate_releases_stub ${root} > releases.yml
    generate_job_templates_stub > job_templates.yml
    generate_env_stub > env.yml
  popd

  pushd "${root}/cf-release"
    spiff merge \
      "${root}/postgres-ci-env/deployments/cf/pgci-cf.yml" \
      "${root}/postgres-ci-env/deployments/common/properties.yml" \
      "${root}/stubs/env.yml" \
      "${root}/postgres-ci-env/deployments/common/common.yml" > "${root}/partial-pgci-cf.yml"

    spiff merge \
      "templates/generic-manifest-mask.yml" \
      "templates/cf.yml" \
      "${root}/postgres-ci-env/deployments/cf/cf-infrastructure-softlayer.yml" \
      "${root}/stubs/releases.yml" \
      "${root}/stubs/job_templates.yml" \
      "${root}/partial-pgci-cf.yml" > "${root}/pgci_cf.yml"
  popd

  bosh -n deploy -d "${CF_DEPLOYMENT}" "${root}/pgci_cf.yml"
}

main "${PWD}"
