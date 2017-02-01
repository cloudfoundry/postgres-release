package deploy_test

import (
	"fmt"
	"testing"

	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
)

var (
	director     boshdir.Director
	configParams helpers.PgatsConfig
)

func TestDeploy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "deploy")
}

var _ = BeforeSuite(func() {
	configPath, err := helpers.ConfigPath()
	Expect(err).NotTo(HaveOccurred())

	configParams, err = helpers.LoadConfig(configPath)
	Expect(err).NotTo(HaveOccurred())

	directorURL := fmt.Sprintf("https://%s:25555", configParams.Bosh.Target)

	director, err = helpers.TargetDirector(directorURL, configParams.Bosh.Username, configParams.Bosh.Password, configParams.Bosh.DirectorCACert)
	Expect(err).NotTo(HaveOccurred())
	//deps, err := director.Deployments()
})
