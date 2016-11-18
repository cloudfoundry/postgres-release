#!/bin/bash -e

DATABASE_URL="postgres://${PG_USER}:${PG_PSW}@${PG_HOST}:${PG_PORT}/${PG_DB}"

function main() {
  psql $DATABASE_URL -c "\l"
  psql $DATABASE_URL -c "\d"
  pgbench $DATABASE_URL -i -s 2
  accounts=$(psql $DATABASE_URL -X -P t -P format=unaligned -c "select count(*) from pgbench_accounts")
  psql $DATABASE_URL -c "\d"
  echo "$accounts accounts present !!"
}

main "${PWD}"
