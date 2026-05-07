#!/usr/bin/env bash
NEED_COMMIT=false

set -euo pipefail

echo "${PRIVATE_YML}" > postgres-release/config/private.yml

get_old_blob_path() {
    local major_version="$1"
    local blobs_file="config/blobs.yml"

    if grep -q "postgresql-${major_version}" "$blobs_file"; then
        cat $blobs_file | grep "postgresql-${major_version}" | cut -f1 -d:
    else
        echo ""
    fi
}

pushd postgres-release
  CURRENT_BLOBS=$(bosh blobs)
  BLOB_PATH=$(ls ../postgres-src/postgresql-*.tar.gz)
  FILENAME=$( basename ${BLOB_PATH} )
  OLD_BLOB_PATH=$(get_old_blob_path "${MAJOR_VERSION}")
  if ! echo "${CURRENT_BLOBS}" | grep "${FILENAME}" ; then
    NEED_COMMIT=true
    echo "adding ${FILENAME}"
    bosh add-blob --sha2 "${BLOB_PATH}" "postgres/${FILENAME}"
    if [[ -n "${OLD_BLOB_PATH}" ]]; then
          bosh remove-blob ${OLD_BLOB_PATH}
    fi
    bosh upload-blobs
  fi

  if ${NEED_COMMIT}; then
    cp ../yq-release/yq_linux_amd64 /tmp/yq && chmod +x /tmp/yq
    sha256_col=$(grep -n "^SHA-256$" ../yq-release/checksums_hashes_order | cut -f1 -d:)
    expected_sha256=$(awk -v col="$((sha256_col + 1))" '/^yq_linux_amd64[[:space:]]/{print $col}' ../yq-release/checksums)
    actual_sha256=$(sha256sum /tmp/yq | awk '{print $1}')
    if [ "$expected_sha256" != "$actual_sha256" ]; then
      echo "SHA-256 verification failed for yq_linux_amd64"
      echo "Expected: $expected_sha256"
      echo "Actual:   $actual_sha256"
      exit 1
    fi
    echo "SHA-256 verified: $actual_sha256"

    echo "-----> $(date): Update the PostgreSQL version inside the used_postgresql_versions.yml file"
    current_minor_version=$(cat config/blobs.yml  | grep "postgresql-${MAJOR_VERSION}" | cut -f1 -d: | sed "s/postgres\/postgresql-//g" | sed "s/.tar.gz//g")
    CURRENT_MINOR_VERSION=$current_minor_version /tmp/yq -i '.postgresql.major_version[env(MAJOR_VERSION)].minor_version = strenv(CURRENT_MINOR_VERSION)' jobs/postgres/templates/used_postgresql_versions.yml

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
