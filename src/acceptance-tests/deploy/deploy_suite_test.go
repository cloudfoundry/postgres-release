package deploy_test

import (
	"testing"

	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	configParams            helpers.PgatsConfig
	deployHelper            helpers.DeployHelper
	latestPostgreSQLVersion string
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

	latestPostgreSQLVersion = configParams.PostgreSQLVersion
	if latestPostgreSQLVersion == "current" {
		versions, err := helpers.NewPostgresReleaseVersions(configParams.VersionsFile)
		Expect(err).NotTo(HaveOccurred())
		latestPostgreSQLVersion = versions.GetPostgreSQLVersion(versions.GetLatestVersion())
	}

	deployHelper, err = helpers.NewDeployHelper(configParams, "fresh", helpers.DeployLatestVersion)
	Expect(err).NotTo(HaveOccurred())

	err = deployHelper.UploadLatestReleaseFromURL("cloudfoundry", "os-conf-release")
	Expect(err).NotTo(HaveOccurred())

	By("Deploying a single postgres instance")
	err = deployHelper.Deploy()
	Expect(err).NotTo(HaveOccurred())
	deployHelper.EnablePrintDiffs()

	By("Populating the database")
	pgprops, pgHost, err := deployHelper.GetPGPropsAndHost()
	Expect(err).NotTo(HaveOccurred())
	db, err := deployHelper.ConnectToPostgres(pgHost, pgprops)
	Expect(err).NotTo(HaveOccurred())
	err = db.CreateAndPopulateTables(pgprops.Databases.Databases[0].Name, helpers.SmallLoad)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	var err error
	err = deployHelper.GetDeployment().DeleteDeployment()
	Expect(err).NotTo(HaveOccurred())
})
