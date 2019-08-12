#!/bin/bash -eu

preflight_check() {
  set +x
  test -n "${BOSH_DIRECTOR_IP}"
  test -n "${BOSH_DIRECTOR_NAME}"
  test -n "${BOSH_CLIENT}"
  test -n "${BOSH_CLIENT_SECRET}"
  test -n "${BOSH_CA_CERT}"
  test -n "${CF_DEPLOYMENT}"
  test -n "${API_PASSWORD}"
  test -n "${USE_LATEST_PGREL}"
  set -x
}

function main(){
  local root="${1}"
  preflight_check
  source ${root}/postgres-release/ci/scripts/configure_for_bosh.sh

  EXTRA_OPS=("")

  if [ "$USE_LATEST_PGREL" == "true" ]; then
    EXTRA_OPS+=("-o \"${root}/postgres-release/ci/templates/use-latest-postgres-release.yml\"")
  fi

  bosh interpolate "${root}/cf-deployment/cf-deployment.yml" \
    -o "${root}/cf-deployment/operations/bosh-lite.yml" \
    -v deployment_name="${CF_DEPLOYMENT}" \
    -v network_name="default" \
    -v system_domain="${BOSH_DIRECTOR_IP}.nip.io" \
    -v cf_admin_password="${API_PASSWORD}" \
    -o "${root}/cf-deployment/operations/use-compiled-releases.yml" \
    -o "${root}/postgres-release/ci/templates/change-router-props-for-boshlite.yml" \
    -o "${root}/postgres-release/ci/templates/add-system-domain-dns-alias.yml" \
    -o "${root}/cf-deployment/operations/use-postgres.yml" \
    -o "${root}/cf-deployment/operations/use-latest-stemcell.yml" \
    ${EXTRA_OPS[@]} \
    > "${root}/pgci_cf.yml"

  bosh --non-interactive --deployment="${CF_DEPLOYMENT}" deploy "${root}/pgci_cf.yml"
}

main "${PWD}"
