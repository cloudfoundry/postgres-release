#!/bin/bash -e
pgversion_upgrade_from=$1
pgversion_current=$2

# From postgres-x.y.z, it's major if x and y are not the same
# in $pgversion_current and $pgversion_upgrade_from

function check_postgresql_versions(){
    if [[ $(echo -e "$pgversion_upgrade_from\n$pgversion_current" | sort --version-sort | head --lines=1) != $pgversion_upgrade_from ]]; then
        echo "The downgrade of the database instance is not supported."
        exit 1
    fi
}

function is_major() {
  [ "${pgversion_current%.*}" != "${pgversion_upgrade_from%.*}" ]
}

if is_major && check_postgresql_versions; then
 echo is major
else
 echo is minor
fi
