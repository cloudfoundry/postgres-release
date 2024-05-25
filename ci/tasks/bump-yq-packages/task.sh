#!/usr/bin/env bash
NEED_COMMIT=false

set -euo pipefail

echo "${PRIVATE_YML}" > postgres-release/config/private.yml

pushd postgres-release
  CURRENT_BLOBS=$(bosh blobs)
  CURRENT_VERSION=$(cat ../yq-release/version)
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
