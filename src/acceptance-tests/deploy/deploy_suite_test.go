package deploy_test

import (
	"os"
	"testing"

	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	configParams            helpers.PgatsConfig
	deployHelper            helpers.DeployHelper
	latestPostgreSQLVersion string
	DB                      helpers.PGData
	pgprops                 helpers.Properties
	pgHost                  string
)

func TestDeploy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "deploy")
}

var _ = BeforeSuite(func() {
	var err error

	configPath, err := helpers.ConfigPath()
	Expect(err).NotTo(HaveOccurred())

	configParams, err = helpers.LoadConfig(configPath)
	Expect(err).NotTo(HaveOccurred())

	deployHelper.Initialize(configParams, "fresh", helpers.DeployLatestVersion)
	Expect(err).NotTo(HaveOccurred())

	err = deployHelper.UploadLatestReleaseFromURL("cloudfoundry", "os-conf-release")
	Expect(err).NotTo(HaveOccurred())

	latestPostgreSQLVersion = configParams.PostgreSQLVersion
	if latestPostgreSQLVersion == "current" {
		versions, err := helpers.NewPostgresReleaseVersions(configParams.VersionsFile)
		Expect(err).NotTo(HaveOccurred())
		latestPostgreSQLVersion = versions.GetPostgreSQLVersion(versions.GetLatestVersion())
	}

	By("Deploying a single postgres instance")
	err = deployHelper.Deploy()
	Expect(err).NotTo(HaveOccurred())

	By("Initializing a postgres client connection")
	pgprops, pgHost, err = deployHelper.GetPGPropsAndHost()
	Expect(err).NotTo(HaveOccurred())
	DB, err = deployHelper.ConnectToPostgres(pgHost, pgprops)
	Expect(err).NotTo(HaveOccurred())

	By("Populating the database")
	err = DB.CreateAndPopulateTables(pgprops.Databases.Databases[0].Name, helpers.SmallLoad)
	Expect(err).NotTo(HaveOccurred())

	By("Validating the database")
	pgData, err := DB.GetData()
	Expect(err).NotTo(HaveOccurred())
	validator := helpers.NewValidator(pgprops, pgData, DB, latestPostgreSQLVersion)
	err = validator.ValidateAll()
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	var err error
	if DB.Data.SSLRootCert != "" {
		err = os.Remove(DB.Data.SSLRootCert)
		Expect(err).NotTo(HaveOccurred())
	}
	if DB.Data.CertUser.Certificate != "" {
		err = os.Remove(DB.Data.CertUser.Certificate)
		Expect(err).NotTo(HaveOccurred())
	}
	if DB.Data.CertUser.Key != "" {
		err = os.Remove(DB.Data.CertUser.Key)
		Expect(err).NotTo(HaveOccurred())
	}
	err = deployHelper.GetDeployment().DeleteDeployment()
	Expect(err).NotTo(HaveOccurred())
})
