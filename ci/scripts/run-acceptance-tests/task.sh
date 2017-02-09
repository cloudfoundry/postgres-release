#!/bin/bash -exu

root="${PWD}"

function create_config_file() {
  indented_cert=$(echo "$BOSH_CA_CERT" | awk '$0="    "$0')
  cat <<EOF
---
bosh:
  target: $BOSH_DIRECTOR
  username: $BOSH_USER
  password: $BOSH_PASSWORD
  director_ca_cert: |+
$indented_cert
cloud_configs:
  default_vm_type: "pgats"
  default_persistent_disk_type: "dp_10G"
postgres_release_version: $REL_VERSION
EOF
}

function main() {
  config_file="${root}/pgats_config.yml"
  create_config_file > $config_file
  to_dir=${GOPATH}/src/github.com/cloudfoundry/postgres-release
  mkdir -p $to_dir
  cp -R ${root}/postgres-release/* $to_dir
  PGATS_CONFIG=$config_file $to_dir/src/acceptance-tests/scripts/test
}

main
