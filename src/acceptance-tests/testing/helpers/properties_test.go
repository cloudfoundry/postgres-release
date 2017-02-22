package helpers_test

import (
	"errors"

	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Properties", func() {
	Describe("Load properties", func() {
		Context("With a valid input and default values", func() {
			var props helpers.Properties

			It("Corretly load all the data", func() {
				var err error
				var data = `
databases:
  databases:
  - citext: true
    name: sandbox
    tag: test
  port: 5524
`
				props, err = helpers.LoadProperties([]byte(data))
				Expect(err).NotTo(HaveOccurred())
				expected := helpers.Properties{
					Databases: helpers.PgProperties{
						Databases: []helpers.PgDBProperties{
							{CITExt: true,
								Name: "sandbox",
								Tag:  "test"},
						},
						Port:                  5524,
						MaxConnections:        500,
						LogLinePrefix:         "%m: ",
						CollectStatementStats: false,
					},
				}
				Expect(props).To(Equal(expected))
			})
		})
		Context("With a valid input and no default values", func() {
			var props helpers.Properties

			BeforeEach(func() {
				var err error
				var data = `
databases:
  address: x.x.x.x
  databases:
  - citext: true
    name: sandbox
    tag: test
  - citext: true
    name: sandbox2
    tag: test
  port: 5524
  roles:
  - name: pgadmin
    password: admin
    tag: admin
  - name: pgadmin2
    password: admin
    tag: admin
  max_connections: 10
  log_line_prefix: "%d"
  collect_statement_statistics: true
  monit_timeout: 120
  additional_config:
    max_wal_senders: 5
    archive_timeout: 1800s
`
				props, err = helpers.LoadProperties([]byte(data))
				Expect(err).NotTo(HaveOccurred())
			})

			It("Correctly load all the data", func() {
				m := make(helpers.PgAdditionalConfigMap)
				m["archive_timeout"] = "1800s"
				m["max_wal_senders"] = 5
				expected := helpers.Properties{
					Databases: helpers.PgProperties{
						Address: "x.x.x.x",
						Databases: []helpers.PgDBProperties{
							{CITExt: true,
								Name: "sandbox",
								Tag:  "test"},
							{CITExt: true,
								Name: "sandbox2",
								Tag:  "test"},
						},
						Port: 5524,
						Roles: []helpers.PgRoleProperties{
							{Password: "admin",
								Name: "pgadmin",
								Tag:  "admin"},
							{Password: "admin",
								Name: "pgadmin2",
								Tag:  "admin"},
						},
						MaxConnections:        10,
						LogLinePrefix:         "%d",
						CollectStatementStats: true,
						MonitTimeout:          120,
						AdditionalConfig:      m,
					},
				}
				Expect(props).To(Equal(expected))

			})
		})
		Context("With a invalid input", func() {
			var props helpers.Properties

			It("Fail to load the an invalid yaml", func() {
				var err error
				props, err = helpers.LoadProperties([]byte("%%%"))
				Expect(err).To(MatchError(ContainSubstring("yaml: could not find expected directive name")))
			})

			It("Fail to load if mandatory props are missing", func() {
				var err error
				props, err = helpers.LoadProperties([]byte("databases: ~"))
				Expect(err).To(MatchError(errors.New(helpers.MissingMandatoryProp)))
			})
		})
	})
})
