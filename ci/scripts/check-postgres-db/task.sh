#!/bin/bash -e

DATABASE_URL="postgres://${PG_USER}:${PG_PSW}@${PG_HOST}:${PG_PORT}/${PG_DB}"

function main() {
  psql $DATABASE_URL -c "\l"
  psql $DATABASE_URL -c "\d"
  accounts=$(psql $DATABASE_URL -X -P t -P format=unaligned -c "select count(*) from pgbench_accounts")
  if [ "$accounts" == "200000" ]; then
    echo "$accounts accounts present !!"
  else
   exit 8
  fi

  psql $DATABASE_URL -c "drop table pgbench_accounts,pgbench_branches,pgbench_history,pgbench_tellers"
}

main "${PWD}"
