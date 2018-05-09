# Hooks

 The `postgres` job has two monit processes that you can use to run custom code:

- `postgres` runs `databases.hooks` before or after PostgreSQL starts or stops.
- `pg_janitor` periodically runs the `janitor.script` and it also takes care of creating roles and database.

If you plan to use these scripts to run custom code, you have to take into consideration that:

- The return code of the `database.hooks` scripts is not propagated to the monit control job
- The output of the hook scripts is logged into:

  - `/var/vcap/sys/log/postgres/hooks.std{out,err}.log`
  - `/var/vcap/sys/log/postgres/janitor{,.err}.log`

- The time spent in `databases.hooks.pre-start` will delay the start of PostgreSQL. In the same way, the time spent in `databases.hooks.pre-stop` will delay the stop of PostgreSQL. This would influence an eventual deployment. For this reason we suggest to avoid long running actions in the hooks and to leverage the `databases.hooks.timeout` property to prevent unexpected delays.
- The following environment variables will be available in the scripts:

  - `DATA_DIR`: the PostgreSQL data directory (e.g. `/var/vcap/store/postgres/postgres-x.x.x`)
  - `PORT`: the PostgreSQL port (e.g. `5432`)
  - `PACKAGE_DIR`: the PostgreSQL binaries directory (e.g. `/var/vcap/packages/postgres-x.x.x`)

  If for example you want to use psql in your hook, you can specify:
  `${PACKAGE_DIR}/bin/psql -p ${PORT} -U vcap postgres -c "\l"`

- In relation to the start up sequence, `databases.hooks.post-start` and `pg_janitor` may run concurrently. It implies that `databases.hooks.post-start` may or may not run before `pg_janitor` actually creates the roles and databases; if you need a database or a role there, wait that it has been actually created before using it.
- Since monit starts and stops postgres based on the PostgreSQL process id, the `databases.hooks.post-stop` script may run concurrently with the restart of PostgreSQL. Likewise, the `databases.hooks.post-start` script may run concurrently with the stop of PostgreSQL.

## Using hooks to replace `run_on_every_startup` property

The `run_on_every_startup` property allowed to run a list of SQL commands at each postgres start against a given database as `vcap`. This property has been removed in postgres-release v29. You can migrate from this property to hooks instead.

Replace:

```yaml
properties:
  databases:
    databases:
    - name: sandbox
      run_on_every_startup:
      - "SQL-QUERY1"
      - "SQL-QUERY2"
```

 with:

```bash
databases:
  hooks:
    post_start: |
      #!/bin/bash
      for i in {10..0}; do
        result=$(${PACKAGE_DIR}/bin/psql -p ${PORT} -U vcap postgres -t -P format=unaligned -c "SELECT 1 from pg_database WHERE datname='sandbox'")
        if [ "$result" == "1" ]; then
          break
        fi
        echo "Database sandbox does not exists yet; trying $i more times"
        sleep 1
      done
      if [ "$i" == "0" ]; then
        echo "Time out waiting for the database to be created"
        exit 1
      fi
      ${PACKAGE_DIR}/bin/psql -p ${PORT} -U vcap sandbox -c "SQL-QUERY1"
      ${PACKAGE_DIR}/bin/psql -p ${PORT} -U vcap sandbox -c "SQL-QUERY2"
```

