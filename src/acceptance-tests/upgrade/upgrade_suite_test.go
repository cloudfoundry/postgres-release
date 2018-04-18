package upgrade_test

import (
	"testing"

	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	configParams helpers.PgatsConfig
	versions     helpers.PostgresReleaseVersions
)

func TestUpgrade(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "upgrade")
}

var _ = BeforeSuite(func() {
	configPath, err := helpers.ConfigPath()
	Expect(err).NotTo(HaveOccurred())

	configParams, err = helpers.LoadConfig(configPath)
	Expect(err).NotTo(HaveOccurred())

	versions, err = helpers.NewPostgresReleaseVersions(configParams.VersionsFile)
	Expect(err).NotTo(HaveOccurred())

	directorHelper, err := helpers.NewDeployHelper(configParams, "", 0)
	Expect(err).NotTo(HaveOccurred())

	err = directorHelper.UploadLatestReleaseFromURL("cloudfoundry", "os-conf-release")
	Expect(err).NotTo(HaveOccurred())
})
