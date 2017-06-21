package deploy_test

import (
	"os"
	"strconv"

	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func createOrUpdateDeployment(postgresReleaseVersion int, manifestPath string, name string) error {
	var err error
	releases := make(map[string]string)
	if postgresReleaseVersion != -1 {
		releases["postgres"] = strconv.Itoa(postgresReleaseVersion)
		err = director.UploadPostgresReleaseFromURL(postgresReleaseVersion)
		if err != nil {
			return err
		}
	}
	err = director.SetDeploymentFromManifest(manifestPath, releases, name)
	if err != nil {
		return err
	}
	if director.GetEnv(name).ContainsVariables() {
		vars := make(map[string]interface{})
		if _, err = director.GetEnv(name).GetVmAddress("postgres"); err != nil {

			vars["postgres_host"] = "1.1.1.1"
			err = director.GetEnv(name).EvaluateTemplate(vars, helpers.EvaluateOptions{})
			if err != nil {
				return err
			}
			err = director.GetEnv(name).CreateOrUpdateDeployment()
			if err != nil {
				return err
			}
		}
		var pgHost string
		pgHost, err = director.GetEnv(name).GetVmDNS("postgres")
		if err != nil {
			pgHost, err = director.GetEnv(name).GetVmAddress("postgres")
			if err != nil {
				return err
			}
		}
		vars["postgres_host"] = pgHost
		err = director.SetDeploymentFromManifest(manifestPath, releases, name)
		if err != nil {
			return err
		}
		err = director.GetEnv(name).EvaluateTemplate(vars, helpers.EvaluateOptions{})
		if err != nil {
			return err
		}
	}
	err = director.GetEnv(name).CreateOrUpdateDeployment()
	if err != nil {
		return err
	}
	return nil
}
func getPostgresJobProps(envName string) (helpers.Properties, error) {
	var err error
	manifestProps, err := director.GetEnv(envName).GetJobsProperties()
	if err != nil {
		return helpers.Properties{}, err
	}
	pgprops := manifestProps.GetJobProperties("postgres")[0]
	return pgprops, nil
}
func connectToPostgres(envName string) (helpers.Properties, helpers.PGData, error) {

	pgprops, err := getPostgresJobProps(envName)
	if err != nil {
		return helpers.Properties{}, helpers.PGData{}, err
	}
	vmaddr, err := director.GetEnv(envName).GetVmAddress("postgres")
	if err != nil {
		return helpers.Properties{}, helpers.PGData{}, err
	}
	var adminRole helpers.PgRoleProperties
	for _, role := range pgprops.Databases.Roles {
		for _, permission := range role.Permissions {
			if permission == "SUPERUSER" {
				adminRole = role
				break
			}
		}
	}
	Expect(adminRole).NotTo(Equal(helpers.PgRoleProperties{}))
	pgc := helpers.PGCommon{
		Address:       vmaddr,
		Port:          pgprops.Databases.Port,
		DefUser:       pgprops.Databases.Roles[0].Name,
		DefPassword:   pgprops.Databases.Roles[0].Password,
		AdminUser:     adminRole.Name,
		AdminPassword: adminRole.Password,
	}
	DB, err := helpers.NewPostgres(pgc)
	if err != nil {
		return helpers.Properties{}, helpers.PGData{}, err
	}
	return pgprops, DB, nil
}

var _ = Describe("Deploy single instance", func() {

	var envName string
	var DB helpers.PGData
	var pgprops helpers.Properties
	var manifestPath, deploymentPrefix string
	var version int
	var latestPostgreSQLVersion string

	JustBeforeEach(func() {
		var err error
		envName = helpers.GenerateEnvName(deploymentPrefix)
		latestPostgreSQLVersion = configParams.PostgreSQLVersion
		if latestPostgreSQLVersion == "current" {
			latestPostgreSQLVersion = versions.GetPostgreSQLVersion(versions.GetLatestVersion())
		}
		By("Deploying a single postgres instance")
		err = createOrUpdateDeployment(version, manifestPath, envName)
		Expect(err).NotTo(HaveOccurred())

		By("Initializing a postgres client connection")
		pgprops, DB, err = connectToPostgres(envName)
		Expect(err).NotTo(HaveOccurred())
		By("Populating the database")
		err = DB.CreateAndPopulateTables(pgprops.Databases.Databases[0].Name, helpers.SmallLoad)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("With a fresh deployment", func() {

		BeforeEach(func() {
			manifestPath = "../testing/templates/postgres_simple.yml"
			version = -1
			deploymentPrefix = "fresh"
		})

		It("Successfully deploys a fresh env", func() {
			pgData, err := DB.GetData()
			Expect(err).NotTo(HaveOccurred())
			validator := helpers.NewValidator(pgprops, pgData, DB, latestPostgreSQLVersion)
			err = validator.ValidateAll()
			Expect(err).NotTo(HaveOccurred())

			By("Enabling SSL")
			manifestPath = "../testing/templates/postgres_simple_ssl.yml"
			err = createOrUpdateDeployment(version, manifestPath, envName)
			Expect(err).NotTo(HaveOccurred())

			pgprops, err := getPostgresJobProps(envName)
			Expect(err).NotTo(HaveOccurred())
			Expect(pgprops.Databases.TLS).NotTo(Equal(helpers.PgTLS{}))
			rootCertPath, err := helpers.WriteFile(pgprops.Databases.TLS.CA)
			Expect(err).NotTo(HaveOccurred())
			certPath, err := helpers.WriteFile(pgprops.Databases.TLS.Certificate)
			Expect(err).NotTo(HaveOccurred())

			By("Checking non-secure connections")
			_, err = DB.GetPostgreSQLVersion()
			if err != nil {
				Expect(err.Error()).NotTo(HaveOccurred())
			}

			By("Checking secure connections")
			err = DB.ChangeSSLMode("verify-full", certPath)
			Expect(err).NotTo(HaveOccurred())
			_, err = DB.GetPostgreSQLVersion()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("x509: certificate signed by unknown authority"))
			err = os.Remove(certPath)
			Expect(err).NotTo(HaveOccurred())

			err = DB.ChangeSSLMode("verify-ca", rootCertPath)
			Expect(err).NotTo(HaveOccurred())
			_, err = DB.GetPostgreSQLVersion()
			if err != nil {
				Expect(err.Error()).NotTo(HaveOccurred())
			}

			err = DB.ChangeSSLMode("verify-full", rootCertPath)
			Expect(err).NotTo(HaveOccurred())
			_, err = DB.GetPostgreSQLVersion()
			if err != nil {
				Expect(err.Error()).NotTo(HaveOccurred())
			}
		})
	})
	Describe("Upgrading an existent env", func() {

		AssertUpgradeSuccessful := func() func() {
			return func() {
				var err error
				By("Validating the database has been deployed as requested")
				pgData, err := DB.GetData()
				Expect(err).NotTo(HaveOccurred())
				validator := helpers.NewValidator(pgprops, pgData, DB, versions.GetPostgreSQLVersion(version))
				err = validator.ValidateAll()
				Expect(err).NotTo(HaveOccurred())

				By("Upgrading to the new release")
				err = createOrUpdateDeployment(-1, manifestPath, director.GetEnv(envName).Deployment.Name())
				Expect(err).NotTo(HaveOccurred())

				By("Validating the database content is still valid after upgrade")
				pgDataAfter, err := DB.GetData()
				Expect(err).NotTo(HaveOccurred())

				tablesEqual := validator.CompareTablesTo(pgDataAfter)
				Expect(tablesEqual).To(BeTrue())

				By("Validating the database has been upgraded as requested")
				validator = helpers.NewValidator(pgprops, pgDataAfter, DB, latestPostgreSQLVersion)
				err = validator.ValidateAll()
				Expect(err).NotTo(HaveOccurred())

				By("Validating the VM can still be restarted")
				err = director.GetEnv(envName).Restart("postgres")
				Expect(err).NotTo(HaveOccurred())

			}
		}

		Context("Upgrading from older version", func() {
			BeforeEach(func() {
				manifestPath = "../testing/templates/postgres_simple_nolinks.yml"
				version = versions.GetOlderVersion()
				deploymentPrefix = "upg-older"
			})
			It("Successfully upgrades from older", AssertUpgradeSuccessful())
		})
		Context("Upgrading from old version", func() {
			BeforeEach(func() {
				manifestPath = "../testing/templates/postgres_simple_nolinks.yml"
				version = versions.GetOldVersion()
				deploymentPrefix = "upg-old"
			})
			It("Successfully upgrades from old", AssertUpgradeSuccessful())
		})
		Context("Upgrading from master version", func() {
			BeforeEach(func() {
				manifestPath = "../testing/templates/postgres_simple_nolinks.yml"
				version = versions.GetLatestVersion()
				deploymentPrefix = "upg-master"
			})
			It("Successfully upgrades from master", AssertUpgradeSuccessful())
		})
	})

	AfterEach(func() {
		var err error
		if DB.Data.SSLRootCert != "" {
			err = os.Remove(DB.Data.SSLRootCert)
			Expect(err).NotTo(HaveOccurred())
		}
		err = director.GetEnv(envName).DeleteDeployment()
		Expect(err).NotTo(HaveOccurred())
	})

})
