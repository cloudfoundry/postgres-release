#!/usr/bin/env bash
set -euo pipefail

touch release-notes/release-notes.md

release_notes=""
pushd postgres-release
  version_number=$(find releases -regex ".*postgres-[0-9]*.yml" | egrep -o "[0-9]+" | sort -n | tail -n 1)

  current_version="$(git show HEAD:config/blobs.yml | grep "/${BLOB}" | grep -Eo "[0-9]+(\.[0-9]+)+")"
  previous_version="$(git show v${version_number}:config/blobs.yml | grep "/${BLOB}" | grep -Eo "[0-9]+(\.[0-9]+)+")"

  if [ "${current_version}" != "${previous_version}" ]; then
    release_notes="### Updates:
* Updates ${BLOB} from ${previous_version} to ${current_version}"
  fi
popd

if [ -z "${release_notes}" ]; then
  echo "${BLOB} has not been updated."
  exit 1
fi

echo "${release_notes}"

echo "${release_notes}" >> release-notes/release-notes.md
