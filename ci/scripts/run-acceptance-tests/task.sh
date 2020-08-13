#!/bin/bash -exu

root="${PWD}"

function create_config_file() {
  indented_cert=$(echo "$BOSH_CA_CERT" | awk '$0="      "$0')
  cat <<EOF
---
bosh:
  target: $BOSH_DIRECTOR_IP
  use_uaa: true
  credentials:
    client: $BOSH_CLIENT
    client_secret: $BOSH_CLIENT_SECRET
    ca_cert: |+
$indented_cert
cloud_configs:
  default_vm_type: "pgats"
  default_persistent_disk_type: "10GB"
postgres_release_version: $REL_VERSION
postgresql_version: "PostgreSQL ${current_version}"
EOF
}

function install_bbr() {
  wget -O bbr.tar "https://github.com/cloudfoundry-incubator/bosh-backup-and-restore/releases/download/v${BBR_VERSION}/bbr-${BBR_VERSION}.tar"
  tar xvf bbr.tar releases/bbr --strip-components=1
  mv bbr /usr/local/bin
}

preflight_check() {
  set +x
  test -n "${BOSH_DIRECTOR_IP}"
  test -n "${BOSH_CLIENT}"
  test -n "${BOSH_CLIENT_SECRET}"
  test -n "${BOSH_CA_CERT}"
  test -n "${REL_VERSION}"
  test -n "${BBR_VERSION}"
  set -x
}

function main() {
  preflight_check
  install_bbr
  cat ${root}/postgres-release/jobs/postgres/templates/pgconfig.sh.erb | grep current_version > ${root}/pgconfig.sh
  source ${root}/pgconfig.sh
  config_file="${root}/pgats_config.yml"
  create_config_file > $config_file
  to_dir=${GOPATH}/src/github.com/cloudfoundry/postgres-release
  mkdir -p $to_dir
  cp -R ${root}/postgres-release/* $to_dir
  go get -u github.com/golang/dep/cmd/dep
  PGATS_CONFIG="$config_file" "$to_dir/src/acceptance-tests/scripts/test-minimal"
}

main
