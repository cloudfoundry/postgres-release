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
    eval $(grep "^current_version=" jobs/postgres/templates/pgconfig.sh.erb)
    new_version=$(echo "${FILENAME}" | egrep -o "[0-9.]+[0-9]")

    # If new version is greater than current version, update the current_version variables
    if printf '%s\n' "$current_version" "$new_version" | sort -V -C; then
      sed -i "s/^current_version=.*/current_version=\"${new_version}\"/" jobs/postgres/templates/pgconfig.sh.erb
      sed -i "s/^current_version=.*/current_version=\"${new_version}\"/" jobs/bbr-postgres-db/templates/config.sh.erb
    fi

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
