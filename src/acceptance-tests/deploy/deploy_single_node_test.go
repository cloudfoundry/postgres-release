package deploy_test

import (
	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func deployEnv(postgresVersion string, manifestPath string, prefix string) (helpers.DeploymentData, error) {
	name := helpers.GenerateEnvName(prefix)
	return updateEnv(postgresVersion, manifestPath, name)
}
func updateEnv(postgresVersion string, manifestPath string, name string) (helpers.DeploymentData, error) {
	var err error
	deployment, err := helpers.InitializeFromManifestAndSetRelease(configParams, manifestPath, director, postgresVersion, name)
	if err != nil {
		return helpers.DeploymentData{}, err
	}
	if postgresVersion != "" {
		err = deployment.UploadReleaseFromURL(postgresVersion)
		if err != nil {
			return helpers.DeploymentData{}, err
		}
	}

	err = deployment.CreateOrUpdateDeployment()
	if err != nil {
		return helpers.DeploymentData{}, err
	}
	return deployment, nil
}
func connectToPostgres(deployment helpers.DeploymentData) (helpers.PgProperties, helpers.PGData, error) {
	var err error
	props, err := deployment.GetPostgresProps()
	if err != nil {
		return helpers.PgProperties{}, helpers.PGData{}, err
	}
	pgprops := props.Databases
	vmaddr, err := deployment.GetVmAddress("postgres")
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

	var deployment helpers.DeploymentData
	var DB helpers.PGData
	var pgprops helpers.PgProperties
	var manifestPath, version, deploymentPrefix string

	const olderVersion = "1"
	const oldVersion = "5"

	JustBeforeEach(func() {
		var err error
		By("Deploying a single postgres instance")
		deployment, err = deployEnv(version, manifestPath, deploymentPrefix)
		Expect(err).NotTo(HaveOccurred())
		By("Initializing a postgres client connection")
		pgprops, DB, err = connectToPostgres(deployment)
		Expect(err).NotTo(HaveOccurred())
		By("Populating the database")
		err = DB.CreateAndPopulateTables(pgprops.Databases[0].Name, helpers.SmallLoad)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("With a fresh deployment", func() {

		BeforeEach(func() {
			manifestPath = "../testing/templates/postgres_simple.yml"
			version = ""
			deploymentPrefix = "fresh"
		})

		It("Successfully deploys a fresh env", func() {
			pgData, err := DB.GetData()
			Expect(err).NotTo(HaveOccurred())
			validator := helpers.NewValidator(pgprops, pgData, DB, "PostgreSQL 9.4.9")
			err = validator.ValidateAll()
			Expect(err).NotTo(HaveOccurred())
		})
	})
	Describe("Upgrading an existent env", func() {

		var oldPostgresVersion string

		AssertUpgradeSuccessful := func() func() {
			return func() {
				var err error
				By("Validating the database has been deployed as requested")
				pgData, err := DB.GetData()
				Expect(err).NotTo(HaveOccurred())
				validator := helpers.NewValidator(pgprops, pgData, DB, oldPostgresVersion)
				err = validator.ValidateAll()
				Expect(err).NotTo(HaveOccurred())

				By("Upgrading to the new release")
				deployment, err = updateEnv("", manifestPath, deployment.Deployment.Name())
				Expect(err).NotTo(HaveOccurred())

				By("Validating the database content is still valid after upgrade")
				pgDataAfter, err := DB.GetData()
				Expect(err).NotTo(HaveOccurred())

				tablesEqual := validator.CompareTablesTo(pgDataAfter)
				Expect(tablesEqual).To(BeTrue())

				By("Validating the database has been upgraded as requested")
				validator = helpers.NewValidator(pgprops, pgDataAfter, DB, "PostgreSQL 9.4.9")
				err = validator.ValidateAll()
				Expect(err).NotTo(HaveOccurred())
			}
		}

		Context("Upgrading from an older version", func() {
			BeforeEach(func() {
				manifestPath = "../testing/templates/postgres_simple.yml"
				version = olderVersion
				deploymentPrefix = "upg-older"
				oldPostgresVersion = "PostgreSQL 9.4.6"
			})
			It("Successfully upgrades from older", AssertUpgradeSuccessful())
		})
		Context("Upgrading from an old version", func() {
			BeforeEach(func() {
				manifestPath = "../testing/templates/postgres_simple.yml"
				version = oldVersion
				deploymentPrefix = "upg-old"
				oldPostgresVersion = "PostgreSQL 9.4.9"
			})
			It("Successfully upgrades from old", AssertUpgradeSuccessful())
		})
	})

	AfterEach(func() {
		var err error
		err = deployment.DeleteDeployment()
		Expect(err).NotTo(HaveOccurred())
	})

})
