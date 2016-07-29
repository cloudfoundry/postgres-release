#!/bin/bash -exu

ROOT="${PWD}"

function deploy_diego() {
  bosh -t $BOSH_DIRECTOR download manifest pgci-cf pgci_cf.yml

  pushd diego-release > /dev/null
    ./scripts/generate-deployment-manifest \
      -c $ROOT/pgci_cf.yml \
      -i $ROOT/postgres-ci-env/deployments/diego/iaas-settings.yml \
      -p $ROOT/postgres-ci-env/deployments/diego/property-overrides.yml \
      -n $ROOT/postgres-ci-env/deployments/diego/instance-count-overrides.yml \
      -v $ROOT/postgres-ci-env/deployments/diego/release-versions.yml \
      > $ROOT/pgci_diego.yml
    netdata=$(cat $ROOT/postgres-ci-env/deployments/diego/dns.yml)
    sed -i "s/.*subnets: null.*/$netdata/g" $ROOT/pgci_diego.yml

  popd > /dev/null

  bosh -n \
    -d pgci_diego.yml \
    -t ${BOSH_DIRECTOR} \
    deploy
}

function upload_release() {
  local release
  release=${1}
  bosh -t ${BOSH_DIRECTOR} upload release https://bosh.io/d/github.com/${release}
}

function main() {
  upload_release "cloudfoundry/cflinuxfs2-rootfs-release"
  upload_release "cloudfoundry/diego-release"
  upload_release "cloudfoundry/garden-linux-release"
  upload_release "cloudfoundry-incubator/etcd-release"

  deploy_diego
}

main
