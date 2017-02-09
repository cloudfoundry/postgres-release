#!/bin/bash -exu

root="${PWD}"

function create_config_file() {

  cat <<EOF
---
bosh:
  target: $BOSH_DIRECTOR
  username: $BOSH_USER
  password: $BOSH_PASSWORD
  director_ca_cert: $BOSH_CA_CERT
cloud_configs:
  default_vm_type: "pgats"
  default_persistent_disk_type: "dp_10G"
postgres_release_version: $RELVERSION
EOF
}

function main() {
  config_file="${root}/pgats_config.yml"
  create_config_file >> $config_file
  PGATS_CONFIG=$config_file $root/postgres-release/src/acceptance-tests/scripts/test
}

main
