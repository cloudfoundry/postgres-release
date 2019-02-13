#!/bin/bash -exu
source /var/vcap/jobs/pgbackrest/config/config.sh
pgbackrest archive-push "$1"
