package helpers_test

import (
	"errors"
	"fmt"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validate deployment", func() {
	var (
		validator helpers.Validator
		mocks     map[string]sqlmock.Sqlmock
	)
	BeforeEach(func() {
		manifestProps := helpers.PgProperties{
			Databases: []helpers.PgDBProperties{
				helpers.PgDBProperties{
					CITExt: true,
					Name:   "db1",
				},
			},
			Roles: []helpers.PgRoleProperties{
				helpers.PgRoleProperties{
					Name:     "pgadmin",
					Password: "admin",
					Permissions: []string{
						"NOSUPERUSER",
						"CREATEDB",
						"CREATEROLE",
						"NOINHERIT",
						"REPLICATION",
						"CONNECTION LIMIT 20",
						"VALID UNTIL 'May 5 12:00:00 2017 +1'",
					},
				},
			},
			Port:                  5522,
			MaxConnections:        30,
			LogLinePrefix:         "xxx",
			CollectStatementStats: true,
			AdditionalConfig: helpers.PgAdditionalConfigMap{
				"max_wal_senders": 5,
				"archive_timeout": "1800s",
			},
		}
		postgresData := helpers.PGOutputData{
			Roles: []helpers.PGRole{
				helpers.PGRole{
					Name: "vcap",
				},
				helpers.PGRole{
					Name:        "pgadmin",
					Super:       false,
					Inherit:     false,
					CreateRole:  true,
					CreateDb:    true,
					CanLogin:    true,
					Replication: true,
					ConnLimit:   20,
					ValidUntil:  "2017-05-05T11:00:00+00:00",
				},
			},
			Databases: []helpers.PGDatabase{
				helpers.PGDatabase{
					Name: "postgres",
					DBExts: []helpers.PGDatabaseExtensions{
						helpers.PGDatabaseExtensions{
							Name: "plpgsql",
						},
					},
				},
				helpers.PGDatabase{
					Name: "db1",
					DBExts: []helpers.PGDatabaseExtensions{
						helpers.PGDatabaseExtensions{
							Name: "pgcrypto",
						},
						helpers.PGDatabaseExtensions{
							Name: "plpgsql",
						},
						helpers.PGDatabaseExtensions{
							Name: "citext",
						},
						helpers.PGDatabaseExtensions{
							Name: "pg_stat_statements",
						},
					},
				},
			},
			Settings: map[string]string{
				"log_line_prefix": "xxx",
				"max_wal_senders": "5",
				"archive_timeout": "1800s",
				"port":            "5522",
				"other":           "other",
				"max_connections": "30",
			},
		}

		mocks = make(map[string]sqlmock.Sqlmock)
		db, mock, err := sqlmock.New()
		Expect(err).NotTo(HaveOccurred())
		mocks["postgres"] = mock
		pg := helpers.PGData{
			Data: helpers.PGCommon{},
			DBs: []helpers.PGConn{
				helpers.PGConn{
					DB:       db,
					TargetDB: "postgres",
				},
			},
		}

		validator = helpers.Validator{
			ManifestProps: manifestProps,
			PostgresData:  postgresData,
			PG:            pg,
		}
	})

	Describe("Validate a good deployment", func() {
		Context("Validate all", func() {
			It("Properly validates dbs", func() {
				err := validator.ValidateDatabases()
				Expect(err).NotTo(HaveOccurred())
			})
			It("Properly validates roles", func() {
				input := "May 5 12:00:00 2017 +1"
				expected := "2017-05-05T11:00:00+00:00"
				err := mockDate(input, expected, mocks)
				Expect(err).NotTo(HaveOccurred())
				err = validator.ValidateRoles()
				Expect(err).NotTo(HaveOccurred())
			})
			It("Properly validates settings", func() {
				err := validator.ValidateSettings()
				Expect(err).NotTo(HaveOccurred())
			})
			It("Properly validates all", func() {
				input := "May 5 12:00:00 2017 +1"
				expected := "2017-05-05T11:00:00+00:00"
				err := mockDate(input, expected, mocks)
				Expect(err).NotTo(HaveOccurred())
				err = validator.ValidateAll()
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
	Describe("Validate a bad deployment", func() {
		Context("Validate databases", func() {
			It("Fails if DB missing", func() {
				validator.ManifestProps.Databases = []helpers.PgDBProperties{
					helpers.PgDBProperties{
						CITExt: true,
						Name:   "db1",
					},
					helpers.PgDBProperties{
						CITExt: true,
						Name:   "zz2",
					},
				}
				err := validator.ValidateDatabases()
				Expect(err).To(MatchError(errors.New(fmt.Sprintf(helpers.MissingDatabaseValidationError, "zz2"))))
			})
			It("Fails if DB extension missing", func() {
				validator.PostgresData.Databases[1].DBExts = []helpers.PGDatabaseExtensions{}
				err := validator.ValidateDatabases()
				Expect(err).To(MatchError(errors.New(fmt.Sprintf(helpers.MissingExtensionValidationError, "pgcrypto", "db1"))))
			})
			It("Fails if extra database present", func() {
				validator.ManifestProps.Databases = []helpers.PgDBProperties{}
				err := validator.ValidateDatabases()
				Expect(err).To(MatchError(errors.New(fmt.Sprintf(helpers.ExtraDatabaseValidationError, "db1"))))
			})
			It("Fails if extra extension present", func() {
				validator.ManifestProps.Databases[0].CITExt = false
				err := validator.ValidateDatabases()
				Expect(err).To(MatchError(errors.New(fmt.Sprintf(helpers.ExtraExtensionValidationError, "citext", "db1"))))
			})
		})
		Context("Validate roles", func() {
			It("Fails if role missing", func() {
				validator.ManifestProps.Roles = []helpers.PgRoleProperties{
					helpers.PgRoleProperties{
						Name:     "pgadmin",
						Password: "admin",
					},
					helpers.PgRoleProperties{
						Name:     "pgadmin2",
						Password: "admin2",
					},
				}
				err := validator.ValidateRoles()
				Expect(err).To(MatchError(errors.New(fmt.Sprintf(helpers.MissingRoleValidationError, "pgadmin2"))))
			})
			It("Fails if extra role present", func() {
				validator.ManifestProps.Roles = []helpers.PgRoleProperties{}
				err := validator.ValidateRoles()
				Expect(err).To(MatchError(errors.New(fmt.Sprintf(helpers.ExtraRoleValidationError, "pgadmin"))))
			})
			It("Fails if incorrect role permission", func() {
				validator.ManifestProps.Roles = []helpers.PgRoleProperties{
					helpers.PgRoleProperties{
						Name:     "pgadmin",
						Password: "admin",
						Permissions: []string{
							"SUPERUSER",
							"CREATEDB",
							"CREATEROLE",
							"NOINHERIT",
							"REPLICATION",
							"CONNECTION LIMIT 21",
							"VALID UNTIL 'May 5 12:00:00 2017 +1'",
						},
					},
				}
				input := "May 5 12:00:00 2017 +1"
				expected := "2017-05-05T11:00:00+00:00"
				err := mockDate(input, expected, mocks)
				Expect(err).NotTo(HaveOccurred())
				err = validator.ValidateRoles()
				Expect(err).To(MatchError(errors.New(fmt.Sprintf(helpers.IncorrectRolePrmissionValidationError, "pgadmin"))))
			})
		})
		Context("Validate settings", func() {
			It("Fails if additional prop value is incorrect", func() {
				validator.ManifestProps.AdditionalConfig["max_wal_senders"] = 10
				err := validator.ValidateSettings()
				Expect(err).To(MatchError(errors.New(fmt.Sprintf(helpers.IncorrectSettingValidationError, "max_wal_senders"))))
			})
			It("Fails if additional prop value is missing", func() {
				validator.ManifestProps.AdditionalConfig["some_prop"] = 10
				err := validator.ValidateSettings()
				Expect(err).To(MatchError(errors.New(fmt.Sprintf(helpers.MissingSettingValidationError, "some_prop"))))
			})
			It("Fails if port is wrong", func() {
				validator.ManifestProps.Port = 1111
				err := validator.ValidateSettings()
				Expect(err).To(MatchError(errors.New(fmt.Sprintf(helpers.IncorrectSettingValidationError, "port"))))
			})
			It("Fails if max connextions setting is wrong", func() {
				validator.ManifestProps.MaxConnections = 10
				err := validator.ValidateSettings()
				Expect(err).To(MatchError(errors.New(fmt.Sprintf(helpers.IncorrectSettingValidationError, "max_connections"))))
			})
			It("Fails if log_line_prefix setting is wrong", func() {
				validator.ManifestProps.LogLinePrefix = "yyy"
				err := validator.ValidateSettings()
				Expect(err).To(MatchError(errors.New(fmt.Sprintf(helpers.IncorrectSettingValidationError, "log_line_prefix"))))
			})
		})
	})
})
