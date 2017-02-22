package deploy_test

import (
	"strconv"

	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func createDeployment(postgresReleaseVersion int, manifestPath string, prefix string) error {
	name := helpers.GenerateEnvName(prefix)
	return updateDeployment(postgresReleaseVersion, manifestPath, name)
}
func updateDeployment(postgresReleaseVersion int, manifestPath string, name string) error {
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
	err = director.DeploymentInfo.CreateOrUpdateDeployment()
	if err != nil {
		return err
	}
	return nil
}
func connectToPostgres() (helpers.PgProperties, helpers.PGData, error) {
	var err error
	props, err := director.DeploymentInfo.GetPostgresProps()
	if err != nil {
		return helpers.PgProperties{}, helpers.PGData{}, err
	}
	pgprops := props.Databases
	vmaddr, err := director.DeploymentInfo.GetVmAddress("postgres")
	if err != nil {
		return helpers.PgProperties{}, helpers.PGData{}, err
	}
	pgc := helpers.PGCommon{
		Address:     vmaddr,
		Port:        pgprops.Port,
		DefUser:     pgprops.Roles[0].Name,
		DefPassword: pgprops.Roles[0].Password,
	}
	DB, err := helpers.NewPostgres(pgc)
	if err != nil {
		return helpers.PgProperties{}, helpers.PGData{}, err
	}
	return pgprops, DB, nil
}

var _ = Describe("Deploy single instance", func() {

	var DB helpers.PGData
	var pgprops helpers.PgProperties
	var manifestPath, deploymentPrefix string
	var version int
	var latestPostgreSQLVersion string

	JustBeforeEach(func() {
		var err error
		latestPostgreSQLVersion = configParams.PostgreSQLVersion
		if latestPostgreSQLVersion == "current" {
			latestPostgreSQLVersion = versions.GetPostgreSQLVersion(versions.GetLatestVersion())
		}
		By("Deploying a single postgres instance")
		err = createDeployment(version, manifestPath, deploymentPrefix)
		Expect(err).NotTo(HaveOccurred())
		By("Initializing a postgres client connection")
		pgprops, DB, err = connectToPostgres()
		Expect(err).NotTo(HaveOccurred())
		By("Populating the database")
		err = DB.CreateAndPopulateTables(pgprops.Databases[0].Name, helpers.SmallLoad)
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
				err = updateDeployment(-1, manifestPath, director.DeploymentInfo.Deployment.Name())
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
			}
		}

		Context("Upgrading from older version", func() {
			BeforeEach(func() {
				manifestPath = "../testing/templates/postgres_simple.yml"
				version = versions.GetOlderVersion()
				deploymentPrefix = "upg-older"
			})
			It("Successfully upgrades from older", AssertUpgradeSuccessful())
		})
		Context("Upgrading from old version", func() {
			BeforeEach(func() {
				manifestPath = "../testing/templates/postgres_simple.yml"
				version = versions.GetOldVersion()
				deploymentPrefix = "upg-old"
			})
			It("Successfully upgrades from old", AssertUpgradeSuccessful())
		})
		Context("Upgrading from master version", func() {
			BeforeEach(func() {
				manifestPath = "../testing/templates/postgres_simple.yml"
				version = versions.GetLatestVersion()
				deploymentPrefix = "upg-master"
			})
			It("Successfully upgrades from master", AssertUpgradeSuccessful())
		})
	})

	AfterEach(func() {
		var err error
		err = director.DeploymentInfo.DeleteDeployment()
		Expect(err).NotTo(HaveOccurred())
	})

})
