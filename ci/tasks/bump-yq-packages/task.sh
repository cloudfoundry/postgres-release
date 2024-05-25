#!/usr/bin/env bash
NEED_COMMIT=false

set -euo pipefail

echo "${PRIVATE_YML}" > postgres-release/config/private.yml

pushd postgres-release
  CURRENT_BLOBS=$(bosh blobs)
  CURRENT_VERSION=$(cat ../yq-src/version)
  mv ../yq-src/yq_linux_amd64 ../yq-src/postgres-yq-${CURRENT_VERSION}
  BLOB_PATH=$(ls ../yq-src/postgres-yq-${CURRENT_VERSION})
  FILENAME=$( basename ${BLOB_PATH} )
  OLD_BLOB_PATH=$(cat config/blobs.yml  | grep "postgres-yq-${MAJOR_VERSION}" | cut -f1 -d:)
  if ! echo "${CURRENT_BLOBS}" | grep "${FILENAME}" ; then
    NEED_COMMIT=true
    echo "adding ${FILENAME}"
    bosh add-blob --sha2 "${BLOB_PATH}" "yq/postgres-${FILENAME}"
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
