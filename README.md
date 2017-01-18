# postgres-release
---

This is a [BOSH](https://www.bosh.io) release for [PostgreSQL](https://www.postgresql.org/).

###Contents

* [Deploying](#deploying)
* [Customizing](#customizing)
* [Contributing](#contributing)
* [Known Limitation](#known-limitation)

## Deploying

In order to deploy the postgres-release you must follow the standard steps for deploying software with BOSH.

1. Install and target a bosh director.
   Please refer to [bosh documentation](http://bosh.io/docs) for instructions on how to do that.
   Bosh-lite specific instructions can be found [here](https://github.com/cloudfoundry/bosh-lite).

1. Install spiff on your dev machine
   Please refer to https://github.com/cloudfoundry-incubator/spiff/releases/

1. Upload the desired stemcell directly to bosh. [bosh.io](http://bosh.io/stemcells) provides a resource to find and download stemcells.
   For bosh-lite:

  ```
  bosh upload stemcell https://bosh.io/d/stemcells/bosh-warden-boshlite-ubuntu-trusty-go_agent
  ```

1. Upload the latest release from [bosh.io](http://bosh.io/releases/github.com/cloudfoundry/postgres-release?all=1):

  ```
  bosh upload release https://bosh.io/d/github.com/cloudfoundry/postgres-release
  ```

  or create and upload a development release:

  ```
  cd ~/workspace/postgres-release
  bosh -n create release --force && bosh -n upload release
  ```

1. Generate the manifest.

   For bosh-lite run:

  ```
  ~/workspace/postgres-release/scripts/generate-bosh-lite-manifest > OUTPUT_MANIFEST_PATH
  ```

  In general, in the postgres-release you are provided with a sample that you can customize by creating your own stubs:

  ```
  ~/workspace/postgres-release/scripts/generate-deployment-manifest \
  -i IAAS-SETTINGS-STUB-PATH \
  -p PROPERTIES-STUB-PATH > OUTPUT_MANIFEST_PATH
  ```

  In the IAAS-SETTINGS-STUB specify:

  ```
  meta:
    stemcell: (( merge || .meta.default_stemcell ))
    default_stemcell:
      name: <STEMCELL_NAME>
      version: <STEMCELL_VERSION>
  compilation:
    cloud_properties: <COMPILATION_CLOUD_PROPS>
  networks:
  - name: default
   <NETWORK_PROPS>
  resource_pools:
  - name: medium
    cloud_properties: <RESPOOL_CLOUD_PROPS>
  ```

  In the PROPERTIES-STUB specify the properties for the postgres job.
  You can refer to the [bosh-lite sample stub](https://github.com/cloudfoundry/postgres-release/blob/master/templates/bosh-lite/properties.yml) for a basic configuration example.

1. Deploy:

  ```
  bosh -d OUTPUT_MANIFEST_PATH deploy
  ```

## Customizing

The table below shows the most significant properties you can use to customize your postgres installation.
The complete list of available properties can be found in the [spec](https://github.com/cloudfoundry/postgres-release/blob/master/jobs/postgres/spec).

Property | Description
-------- | -------------
databases.port | The database port
databases.databases | A list of databases and associated properties to create when Postgres starts
databases.databases[n].name | Database name
databases.databases[n].citext | If `true` the citext extension is created for the db
databases.databases[n].run_on_every_startup | A list of SQL commands run at each postgres start against the given database as `vcap`
databases.roles | A list of database roles and associated properties to create
databases.roles[n].name | Role name
databases.roles[n].password | Login password for the role
databases.roles[n].permissions| A list of attributes for the role. For the complete list of attributes, refer to [ALTER ROLE command options](https://www.postgresql.org/docs/9.4/static/sql-alterrole.html).
databases.max_connections | Maximum number of database connections
databases.log_line_prefix | The postgres `printf` style string that is output at the beginning of each log line. Default: `%m:`
databases.collect_statement_statistics | Enable the `pg_stat_statements` extension and collect statement execution statistics. Default: `false`
databases.additional_config | A map of additional key/value pairs to include as extra configuration properties
databases.monit_timeout | Monit timout in seconds for the postgres job start. By default the global monit timeout applies. You may need to specify a higher value if you have a large database and the postgres-release deployment includes a PostgreSQL upgrade.

*Note*
- Removing a database from `databases.databases` list and deploying again does not trigger a physical deletion of the database in PostgreSQL.
- Removing a role from `databases.roles` list and deploying again does not trigger a physical deletion of the role in PostgreSQL.

## Contributing

### Contributor License Agreement

Contributors must sign the Contributor License Agreement before their
contributions can be merged. Follow the directions
[here](https://www.cloudfoundry.org/community/contribute/) to complete
that process.

### Developer Workflow

1. [Fork](https://help.github.com/articles/fork-a-repo) the repository and make a local [clone](https://help.github.com/articles/fork-a-repo#step-2-create-a-local-clone-of-your-fork)
1. Create a feature branch from the development branch

   ```bash
   cd postgres-release
   git checkout develop
   git checkout -b feature-branch
   ```

Make sure that you are working against the `develop` branch. PRs submitted against other branches will need to be resubmitted with the correct branch targeted.

## Known Limitations

The postgres-release does not directly support high availability.
Even if you deploy more instances, no replication is configured.
