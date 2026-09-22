#!/usr/bin/env bash
NEED_COMMIT=false

set -euo pipefail

echo "${PRIVATE_YML}" > postgres-release/config/private.yml

get_old_blob_path() {
  local blobs_file="config/blobs.yml"
  if grep -q "^icu/" "$blobs_file"; then
    grep "^icu/" "$blobs_file" | head -1 | cut -f1 -d:
  else
    echo ""
  fi
}

pushd postgres-release
  CURRENT_BLOBS=$(bosh blobs)
  BLOB_PATH=$(echo ../icu-src/icu4c-*-src.tgz)
  FILENAME=$(basename "${BLOB_PATH}")
  OLD_BLOB_PATH=$(get_old_blob_path)

  if ! echo "${CURRENT_BLOBS}" | grep -q "${FILENAME}"; then
    NEED_COMMIT=true
    echo "adding ${FILENAME}"
    bosh add-blob --sha2 "${BLOB_PATH}" "icu/${FILENAME}"
    if [[ -n "${OLD_BLOB_PATH}" ]]; then
      bosh remove-blob "${OLD_BLOB_PATH}"
    fi
    bosh upload-blobs
  fi

  if [ "${NEED_COMMIT}" = "true" ]; then
    echo "-----> $(date): Creating git commit"
    git config user.name "$GIT_USER_NAME"
    git config user.email "$GIT_USER_EMAIL"
    git add .

    git --no-pager diff --cached
    if [[ "$( git status --porcelain )" != "" ]]; then
      git commit -am "Bump packages"
    fi
  fi
popd
