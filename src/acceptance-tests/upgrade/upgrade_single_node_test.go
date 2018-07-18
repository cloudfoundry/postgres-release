package upgrade_test

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Upgrading postgres-release", func() {

	var DB helpers.PGData
	var pgprops helpers.Properties
	var version int
	var latestPostgreSQLVersion string
	var pgHost string
	var deploymentPrefix string
	var deployHelper helpers.DeployHelper

	BeforeEach(func() {
		var err error
		deployHelper, err = helpers.NewDeployHelper(configParams, "upgrade", helpers.DeployLatestVersion)
		Expect(err).NotTo(HaveOccurred())
	})

	JustBeforeEach(func() {
		var err error
		deployHelper.SetDeploymentName(deploymentPrefix)
		deployHelper.SetPGVersion(version)
		latestPostgreSQLVersion = configParams.PostgreSQLVersion
		if latestPostgreSQLVersion == "current" {
			latestPostgreSQLVersion = versions.GetPostgreSQLVersion(versions.GetLatestVersion())
		}
		deployHelper.InitializeVariables()

		By("Deploying a single postgres instance")
		err = deployHelper.Deploy()
		Expect(err).NotTo(HaveOccurred())
		deployHelper.EnablePrintDiffs()

		By("Initializing a postgres client connection")
		pgprops, pgHost, err = deployHelper.GetPGPropsAndHost()
		Expect(err).NotTo(HaveOccurred())
		DB, err = deployHelper.ConnectToPostgres(pgHost, pgprops)
		Expect(err).NotTo(HaveOccurred())
		By("Populating the database")
		err = DB.CreateAndPopulateTables(pgprops.Databases.Databases[0].Name, helpers.SmallLoad)
		Expect(err).NotTo(HaveOccurred())
	})

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
			deployHelper.SetPGVersion(helpers.DeployLatestVersion)
			err = deployHelper.Deploy()
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
			err = deployHelper.GetDeployment().Restart("postgres")
			Expect(err).NotTo(HaveOccurred())

			if deploymentPrefix == "upg-old-nocopy" {
				By("Validating the postgres-previous is not created")
				if !versions.IsMajor(latestPostgreSQLVersion, versions.GetOldVersion()) {
					sshKeyFile, err := deployHelper.WriteSSHKey()
					Expect(err).NotTo(HaveOccurred())
					cmd := exec.Command("ssh", "-i", sshKeyFile, "-o", "BatchMode=yes", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), "sudo test -d /var/vcap/store/postgres/postgres-previous")

					stdout, stderr, err := helpers.RunCommand(cmd)
					Expect(err).To(HaveOccurred(), "stderr was: '%v', stdout was: '%v'", stderr, stdout)
					err = os.Remove(sshKeyFile)
					Expect(err).NotTo(HaveOccurred())
				}
			} else if deploymentPrefix == "upg-old" {
				By("Validating the postgres-previous is created")
				sshKeyFile, err := deployHelper.WriteSSHKey()
				Expect(err).NotTo(HaveOccurred())
				cmd := exec.Command("ssh", "-i", sshKeyFile, "-o", "BatchMode=yes", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), "sudo test -d /var/vcap/store/postgres/postgres-previous")

				stdout, stderr, err := helpers.RunCommand(cmd)
				Expect(err).NotTo(HaveOccurred(), "stderr was: '%v', stdout was: '%v'", stderr, stdout)

				err = os.Remove(sshKeyFile)
				Expect(err).NotTo(HaveOccurred())
			}
		}
	}

	Context("Upgrading from minor-no-copy version", func() {

		BeforeEach(func() {
			version = versions.GetOldVersion()
			deploymentPrefix = "upg-old-nocopy"
			deployHelper.SetOpDefs(helpers.Define_upgrade_no_copy_ops())
		})

		It("Successfully upgrades from old with no copy of the data directory", AssertUpgradeSuccessful())
	})

	Context("Upgrading from old version", func() {

		BeforeEach(func() {
			version = versions.GetOldVersion()
			deploymentPrefix = "upg-old"
		})

		It("Successfully upgrades from old", AssertUpgradeSuccessful())
	})

	Context("Upgrading from master version", func() {

		BeforeEach(func() {
			version = versions.GetLatestVersion()
			deploymentPrefix = "upg-master"
		})

		It("Successfully upgrades from master", AssertUpgradeSuccessful())
	})

	AfterEach(func() {
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
})
