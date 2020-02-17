# Postgres-release Acceptance Tests (PGATS)

The acceptance tests run several deployments of the postgres-release in order to exercise a variety of scenario:

- Verify that customizable configurations are properly reflected in the PostgreSQL server
  - Roles
  - Databases
  - Database extensions
  - Properties (e.g. max_connections)
- Test supported upgrade paths from previous versions
- Test ssl support, backup and restore, and hooks

## Get the code

```bash
$ go get github.com/cloudfoundry/postgres-release
$ cd $GOPATH/src/github.com/cloudfoundry/postgres-release
```

## Environment setup

* Upload to the BOSH director the latest stemcell and your dev postgres-release:

  ```bash
  $ bosh upload-stemcell STEMCELL_URL_OR_PATH_TO_DOWNLOADED_STEMCELL
  $ bosh create-release --force
  $ bosh upload-release
  ```

* The acceptance tests are written in Go. Make sure that:
  - Golang (>=1.7) is installed on the machine
  - the postgres-release is inside your $GOPATH

* Some test cases make use of [bbr](https://docs.cloudfoundry.org/bbr/installing.html). Make sure that it is available in your $PATH.

* Go dependencies are managed using [dep](https://golang.github.io/dep/docs/installation.html). Make sure that it is installed.

If you are **not** using BOSH Lite according to the [quick start](http://bosh.io/docs/quick-start/) documentation, note that

* PGATS must have access to the target BOSH director and to the postgres VM deployed from it

* the BOSH director must be configured with the [cloud_config.yml](https://bosh.io/docs/cloud-config.html#update)

* the director must be configured with verifiable [certificates](https://bosh.io/docs/director-certs.html) because PGATS use the `bosh-cli` director package for programmatic access to the Director API.

## Configuration

An example config file for bosh-lite would look like:

```bash
$ cat > $GOPATH/src/github.com/cloudfoundry/postgres-release/pgats_config.yml << EOF
---
bosh:
  target: 192.168.50.6
  credentials:
    client: admin
    # bosh interpolate creds.yml --path /admin_password
    client_secret: admin
    # insert CA cert, e.g. from creds.yml
    # bosh interpolate creds.yml --path /director_ssl/ca
    ca_cert: |+
      -----BEGIN CERTIFICATE-----
      -----END CERTIFICATE-----
  use_uaa: true
cloud_configs:
  default_azs: [z1]
  default_networks:
  - name: default
  default_persistent_disk_type: 10GB
  default_vm_type: small
EOF
```

The full set of config parameters is explained below.

`bosh`parameters are used to connect to the BOSH director that would host the test environment:

* `bosh.target` (required) Public BOSH director ip address
* `bosh.use_uaa` (required) Set to true if the BOSH director is configured to delegate user management to the UAA server.
* `bosh.credentials.client` (required) Username for the BOSH director login
* `bosh.credentials.client_secret` (required) Password for the BOSH director login
* `bosh.credentials.ca_cert` (required) BOSH director CA Cert

`cloud_config` parameters are used to generate a BOSH v2 manifest that matches your IaaS configuration:

* `cloud_config.default_azs` List of vailability zones. It defaults to `[z1]`.
* `cloud_config.default_networks` List of networks. It defaults to `[{name: default}]`.
* `cloud_config.default_persistent_disk_type` Persistent disk type. It defaults to `10GB`.
* `cloud_config.default_vm_type` VM type. It defaults to `small`.

Other paramaters:

* `postgres_release_version` The postgres-release version to test. If not specified, the latest uploaded to the director is used.
* `postgresql_version` The PostgreSQL version that is expected to be deployed. You only need to specify it if your changes include a PostgreSQL version upgrade.
If not specified, we expect that the one in the latest published postgres-release is deployed.

## Running

Run all the tests with:

```bash
$ export PGATS_CONFIG=$GOPATH/src/github.com/cloudfoundry/postgres-release/pgats_config.yml
$ $GOPATH/src/github.com/cloudfoundry/postgres-release/src/acceptance-tests/scripts/test
```

Run a specific set of tests with:

```bash
$ export PGATS_CONFIG=$GOPATH/src/github.com/cloudfoundry/postgres-release/pgats_config.yml
$ $GOPATH/src/github.com/cloudfoundry/postgres-release/src/acceptance-tests/scripts/test <some test packages>
```

The `PGATS_CONFIG` environment variable must point to the absolute path of the [configuration file](#configuration).
