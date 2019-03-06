#!/bin/bash -exu
source /var/vcap/jobs/pgbackrest/config/config.sh
su - vcap -c "${JOB_DIR}/bin/backup.sh"
