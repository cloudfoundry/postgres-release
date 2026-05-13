#!/usr/bin/env bash
NEED_COMMIT=false

set -euo pipefail

echo "${PRIVATE_YML}" > postgres-release/config/private.yml

pushd postgres-release
  CURRENT_BLOBS=$(bosh blobs)
  CURRENT_VERSION=$(cat ../yq-release/version)

  sha256_col=$(grep -n "^SHA-256$" ../yq-release/checksums_hashes_order | cut -f1 -d:)
  expected_sha256=$(awk -v col="$((sha256_col + 1))" '/^yq_linux_amd64[[:space:]]/{print $col}' ../yq-release/checksums)
  actual_sha256=$(sha256sum ../yq-release/yq_linux_amd64 | awk '{print $1}')
  if [ "$expected_sha256" != "$actual_sha256" ]; then
    echo "SHA-256 verification failed for yq_linux_amd64"
    echo "Expected: $expected_sha256"
    echo "Actual:   $actual_sha256"
    exit 1
  fi
  echo "SHA-256 verified: $actual_sha256"
  cp ../yq-release/yq_linux_amd64 ../yq-bin/yq && chmod +x ../yq-bin/yq

  mv ../yq-release/yq_linux_amd64 ../yq-release/postgres-yq-${CURRENT_VERSION}
  BLOB_PATH=$(ls ../yq-release/postgres-yq-${CURRENT_VERSION})
  FILENAME=$( basename ${BLOB_PATH} )
  OLD_BLOB_PATH=$(cat config/blobs.yml  | grep "postgres-yq" | cut -f1 -d:)
  if ! echo "${CURRENT_BLOBS}" | grep "${FILENAME}" ; then
    NEED_COMMIT=true
    echo "adding ${FILENAME}"
    bosh add-blob --sha2 "${BLOB_PATH}" "yq/${FILENAME}"
    bosh remove-blob ${OLD_BLOB_PATH}
    bosh upload-blobs
  fi

  if ${NEED_COMMIT}; then
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
