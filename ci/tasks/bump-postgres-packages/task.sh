#!/usr/bin/env bash
NEED_COMMIT=false

set -euo pipefail

echo "${PRIVATE_YML}" > postgres-release/config/private.yml

pushd postgres-release
  CURRENT_BLOBS=$(bosh blobs)
  BLOB_PATH=$(ls ../postgres-src/postgresql-*.tar.gz)
  FILENAME=$( basename ${BLOB_PATH} )
  OLD_BLOB_PATH=$(cat config/blobs.yml  | grep "postgresql-${MAJOR_VERSION}" | cut -f1 -d:)
  if ! echo "${CURRENT_BLOBS}" | grep "${FILENAME}" ; then
    NEED_COMMIT=true
    echo "adding ${FILENAME}"
    bosh add-blob --sha2 "${BLOB_PATH}" "postgres/${FILENAME}"
    bosh remove-blob ${OLD_BLOB_PATH}
    bosh upload-blobs
  fi

  if ${NEED_COMMIT}; then
    latest_yq_version=$(curl -s -L https://api.github.com/repos/mikefarah/yq/releases/latest | grep "tag_name" | sed s/\"tag_name\":\//g | sed s/\"//g | sed s/\,//g | sed s/v//g | xargs)
    curl -s -L https://github.com/mikefarah/yq/releases/download/v${latest_yq_version}/yq_linux_amd64 -o /tmp/yq && chmod +x /tmp/yq

    echo "-----> $(date): Update the PostgreSQL version inside the used_postgresql_versions.yml file"
    current_minor_version=$(cat config/blobs.yml  | grep "postgresql-${MAJOR_VERSION}" | cut -f1 -d: | sed "s/postgres\/postgresql-//g" | sed "s/.tar.gz//g")
    CURRENT_MINOR_VERSION=$current_minor_version /tmp/yq -i '.postgresql.major_version[env(MAJOR_VERSION)].minor_version = strenv(CURRENT_MINOR_VERSION)' jobs/postgres/config/used_postgresql_versions.yml

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
