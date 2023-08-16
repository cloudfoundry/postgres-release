#!/usr/bin/env bash

set -eu

dir="$(dirname "$0")"

fly -t "${CONCOURSE_TARGET:-bosh}" set-pipeline -p postgres-release -c "$dir/pipeline.yml"
