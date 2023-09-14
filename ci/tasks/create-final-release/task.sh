#!/bin/bash -exu

root_dir="${PWD}"

cd postgres-release

set +x
echo "$RELEASE_PRIVATE_YML" >> "config/private.yml"
set -x

bosh -n create-release --final
new_release_version="$(find releases -regex ".*postgres-[0-9]*.yml" | egrep -o "[0-9]+" | sort -n | tail -n 1)"

eval $(grep "^current_version=" jobs/postgres/templates/pgconfig.sh.erb)
sed -i "/^versions:/a\ \ ${new_release_version}: \"PostgreSQL ${current_version}\"" versions.yml
git add versions.yml

git add .final_builds releases

git config user.name "$GIT_USER_NAME"
git config user.email "$GIT_USER_EMAIL"
git commit -m "Final release ${new_release_version}"

echo "v${new_release_version}" > version_number
