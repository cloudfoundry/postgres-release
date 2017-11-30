#!/bin/bash -eu

preflight_check() {
  set +x
  test -n "${BOSH_DIRECTOR}"
  test -n "${BOSH_PUBLIC_IP}"
  test -n "${BOSH_CLIENT}"
  test -n "${BOSH_CLIENT_SECRET}"
  test -n "${BOSH_CA_CERT}"
  test -n "${CF_DEPLOYMENT}"
  test -n "${API_PASSWORD}"
  test -n "${S3_ACCESS_KEY}"
  test -n "${S3_SECRET_KEY}"
  test -n "${S3_HOST}"
  test -n "${USE_LATEST_PGREL}"
  set -x
}

function main(){
  local root="${1}"
  preflight_check

  export BOSH_ENVIRONMENT="https://${BOSH_DIRECTOR}:25555"

  EXTRA_OPS="-o \"${root}/postgres-release/ci/templates/use-latest-postgres-release.yml\""
  if [ "$USE_LATEST_PGREL" == "false" ]; then
    EXTRA_OPS=" "
  fi

  sed -i -e "s/region: ((aws_region))/host: ((blobstore_s3_host))/g" "${root}/cf-deployment/operations/use-s3-blobstore.yml"

  bosh interpolate "${root}/cf-deployment/cf-deployment.yml" \
    --vars-store "${root}/cf-variables/variables_output.yml" \
    -o "${root}/cf-deployment/operations/rename-deployment.yml" \
    -v deployment_name="${CF_DEPLOYMENT}" \
    -v system_domain="apps.${CF_DEPLOYMENT}.microbosh" \
    -v cf_admin_password="${API_PASSWORD}" \
    -o "${root}/cf-deployment/operations/use-postgres.yml" \
    -o "${root}/cf-deployment/operations/scale-to-one-az.yml" \
    -o "${root}/cf-deployment/operations/use-latest-stemcell.yml" \
    $EXTRA_OPS \
    -v blobstore_access_key_id="${S3_ACCESS_KEY}" \
    -v blobstore_secret_access_key="${S3_SECRET_KEY}" \
    -v blobstore_s3_host="${S3_HOST}" \
    -o "${root}/cf-deployment/operations/use-s3-blobstore.yml" \
    -v buildpack_directory_key="${CF_DEPLOYMENT}-cc-buildpacks" \
    -v droplet_directory_key="${CF_DEPLOYMENT}-cc-droplets" \
    -v app_package_directory_key="${CF_DEPLOYMENT}-cc-app-package" \
    -v resource_directory_key="${CF_DEPLOYMENT}-cc-resource" \
    -o "${root}/postgres-release/ci/templates/change_uaa_props.yml" \
    > "${root}/pgci_cf.yml"

  bosh --non-interactive --deployment="${CF_DEPLOYMENT}" deploy "${root}/pgci_cf.yml"
}

main "${PWD}"
