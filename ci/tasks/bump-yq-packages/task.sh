#!/usr/bin/env bash
NEED_COMMIT=false

set -euo pipefail

echo "${PRIVATE_YML}" > postgres-release/config/private.yml

pushd postgres-release
  CURRENT_BLOBS=$(bosh blobs)
  CURRENT_VERSION=$(cat ../yq-release/version)

  sha256_col=$(grep -n "^SHA-256$" ../yq-release/checksums_hashes_order | cut -f1 -d:)

  for ARCH in amd64 arm64; do
    src_file="yq_linux_${ARCH}"
    blob_name="postgres-yq-${ARCH}-${CURRENT_VERSION}"

    expected_sha256=$(awk -v col="$((sha256_col + 1))" "/^${src_file}[[:space:]]/{print \$col}" ../yq-release/checksums)
    actual_sha256=$(sha256sum "../yq-release/${src_file}" | awk '{print $1}')
    if [ "$expected_sha256" != "$actual_sha256" ]; then
      echo "SHA-256 verification failed for ${src_file}"
      echo "Expected: $expected_sha256"
      echo "Actual:   $actual_sha256"
      exit 1
    fi
    echo "SHA-256 verified for ${src_file}: $actual_sha256"

    cp "../yq-release/${src_file}" "../yq-release/${blob_name}"

    if ! echo "${CURRENT_BLOBS}" | grep -q "${blob_name}"; then
      NEED_COMMIT=true
      echo "adding ${blob_name}"
      bosh add-blob --sha2 "../yq-release/${blob_name}" "yq/${blob_name}"
    fi
  done

  # Remove any old yq blobs that are not the current amd64 or arm64 blobs
  while IFS= read -r old_blob; do
    echo "removing old blob: ${old_blob}"
    bosh remove-blob "${old_blob}"
    NEED_COMMIT=true
  done < <(grep "^yq/" config/blobs.yml \
    | grep -vF "postgres-yq-amd64-${CURRENT_VERSION}" \
    | grep -vF "postgres-yq-arm64-${CURRENT_VERSION}" \
    | cut -f1 -d:)

  if [ "${NEED_COMMIT}" = "true" ]; then
    bosh upload-blobs
  fi

  # Always populate yq-bin from amd64 binary (CI workers are x86_64)
  cp ../yq-release/yq_linux_amd64 ../yq-bin/yq && chmod +x ../yq-bin/yq

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