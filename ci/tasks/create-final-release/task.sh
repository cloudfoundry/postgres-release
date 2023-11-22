#!/bin/bash -exu

root_dir="${PWD}"

cd postgres-release

set +x
echo "$RELEASE_PRIVATE_YML" >> "config/private.yml"
set -x

latest_yq_version=$(curl -s -L https://api.github.com/repos/mikefarah/yq/releases/latest | grep "tag_name" | sed s/\"tag_name\":\//g | sed s/\"//g | sed s/\,//g | sed s/v//g | xargs)
curl -s -L https://github.com/mikefarah/yq/releases/download/v${latest_yq_version}/yq_linux_amd64 -o /tmp/yq && chmod +x /tmp/yq

bosh -n create-release --final
new_release_version="$(find releases -regex ".*postgres-[0-9]*.yml" | egrep -o "[0-9]+" | sort -n | tail -n 1)"

current_major_version=$(/tmp/yq '.postgresql.default' jobs/postgres/config/used_postgresql_versions.yml)
current_minor_version=$(CURRENT_MAJOR_VERSION=$current_major_version /tmp/yq '.postgresql.major_version[env(CURRENT_MAJOR_VERSION)].minor_version' jobs/postgres/config/used_postgresql_versions.yml)
sed -i "/^versions:/a\ \ ${new_release_version}: \"PostgreSQL ${current_minor_version}\"" versions.yml
git add versions.yml

git add .final_builds releases

git config user.name "$GIT_USER_NAME"
git config user.email "$GIT_USER_EMAIL"
git commit -m "Final release ${new_release_version}"

echo "v${new_release_version}" > version_number
