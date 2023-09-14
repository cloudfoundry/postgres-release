#!/bin/bash -xu

root="${PWD}"

function setup_bosh() {
  source start-bosh
  source /tmp/local-bosh/director/env
  bosh -n update-runtime-config \
    --name dns \
    "/usr/local/bosh-deployment/runtime-configs/dns.yml"
}

function create_config_file() {
  indented_cert=$(cat "$BOSH_CA_CERT" | awk '$0="      "$0')
  cat <<EOF
---
bosh:
  target: ${BOSH_ENVIRONMENT}
  use_uaa: true
  credentials:
    client: ${BOSH_CLIENT}
    client_secret: ${BOSH_CLIENT_SECRET}
    ca_cert: |+
$indented_cert
cloud_configs:
  default_vm_type: "default"
  default_persistent_disk_type: "default"
postgresql_version: "PostgreSQL ${current_version}"
EOF
}

function install_bbr() {
  tar xvf bbr-github-release/bbr-*.tar --strip-components=2 ./releases/bbr
  mv bbr /usr/local/bin
}

function upload_release() {
  pushd ${root}/postgres-release
    bosh create-release --force
    bosh upload-release
  popd
}

function main() {
  setup_bosh
  install_bbr
  upload_release
  bosh upload-stemcell stemcell/stemcell.tgz
  cat ${root}/postgres-release/jobs/postgres/templates/pgconfig.sh.erb | grep current_version > ${root}/pgconfig.sh
  source ${root}/pgconfig.sh
  config_file="${root}/pgats_config.yml"
  create_config_file > $config_file
  PGATS_CONFIG="$config_file" postgres-release/src/acceptance-tests/scripts/test
}

main
