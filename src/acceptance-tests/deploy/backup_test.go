package deploy_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backup and restore a deployment", func() {

	var caCertPath string
	var tempDir string
	var pwd string

	JustBeforeEach(func() {
		var err error
		err = deployHelper.Deploy()
		Expect(err).NotTo(HaveOccurred())

		caCertPath, err = helpers.WriteFile(configParams.Bosh.DirectorCACert)
		Expect(err).NotTo(HaveOccurred())

		tempDir, err = helpers.CreateTempDir()
		Expect(err).NotTo(HaveOccurred())
		pwd, err = os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		err = os.Chdir(tempDir)
		Expect(err).NotTo(HaveOccurred())

		os.Setenv("BOSH_CLIENT_SECRET", configParams.Bosh.Password)
		os.Setenv("CA_CERT", caCertPath)
	})

	AfterEach(func() {
		err := os.Chdir(pwd)
		Expect(err).NotTo(HaveOccurred())
		os.Remove(caCertPath)
		os.RemoveAll(tempDir)
	})

	Context("BBR is disabled", func() {

		BeforeEach(func() {
			deployHelper.SetOpDefs(nil)
		})

		It("Fails to run pre-backup-checks", func() {
			var err error
			var cmd *exec.Cmd
			By("Running pre-backup-checks")
			cmd = exec.Command("bbr", "deployment", "--target", configParams.Bosh.Target, "--username", configParams.Bosh.Username, "--deployment", deployHelper.GetDeploymentName(), "pre-backup-check")
			err = cmd.Run()
			Expect(err).To(HaveOccurred())
		})

		It("Fails to backup the database", func() {
			var err error
			var cmd *exec.Cmd
			cmd = exec.Command("bbr", "deployment", "--target", configParams.Bosh.Target, "--username", configParams.Bosh.Username, "--deployment", deployHelper.GetDeploymentName(), "backup")
			err = cmd.Run()
			Expect(err).To(HaveOccurred())
		})

		It("Fails to restore the database", func() {
			var err error
			var cmd *exec.Cmd
			cmd = exec.Command("bbr", "deployment", "--target", configParams.Bosh.Target, "--username", configParams.Bosh.Username, "--deployment", deployHelper.GetDeploymentName(), "restore", "--artifact-path", fmt.Sprintf("%s/doesnotexist", tempDir))
			err = cmd.Run()
			Expect(err).To(HaveOccurred())
		})

	})

	Context("BBR is enabled", func() {

		var pgprops helpers.Properties
		var pgHost string
		var db helpers.PGData

		BeforeEach(func() {
			var err error

			deployHelper.SetOpDefs(helpers.Define_bbr_ops())
			pgprops, pgHost, err = deployHelper.GetPGPropsAndHost()
			Expect(err).NotTo(HaveOccurred())
			db, err = deployHelper.ConnectToPostgres(pgHost, pgprops)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Fails to restore the database", func() {
			var err error
			var cmd *exec.Cmd
			cmd = exec.Command("bbr", "deployment", "--target", configParams.Bosh.Target, "--username", configParams.Bosh.Username, "--deployment", deployHelper.GetDeploymentName(), "restore", "--artifact-path", fmt.Sprintf("%s/doesnotexist", tempDir))
			err = cmd.Run()
			Expect(err).To(HaveOccurred())
		})

		It("Successfully backup and restore the database", func() {
			var err error
			var cmd *exec.Cmd
			By("Running pre-backup-checks")
			cmd = exec.Command("bbr", "deployment", "--target", configParams.Bosh.Target, "--username", configParams.Bosh.Username, "--deployment", deployHelper.GetDeploymentName(), "pre-backup-check")
			err = cmd.Run()
			Expect(err).NotTo(HaveOccurred(), "Check the bbr logfile bbr-TIMESTAMP.err.log for why this has failed")

			By("Changing content")
			err = db.CreateAndPopulateTablesWithPrefix(pgprops.Databases.Databases[0].Name, helpers.Test1Load, "restore")
			Expect(err).NotTo(HaveOccurred())
			result, err := db.CheckTableExist("restore_0", pgprops.Databases.Databases[0].Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeTrue())

			By("Running backup")
			cmd = exec.Command("bbr", "deployment", "--target", configParams.Bosh.Target, "--username", configParams.Bosh.Username, "--deployment", deployHelper.GetDeploymentName(), "backup")
			err = cmd.Run()
			Expect(err).NotTo(HaveOccurred(), "Check the bbr logfile bbr-TIMESTAMP.err.log for why this has failed")
			tarBackupFile := fmt.Sprintf("%s/%s*/postgres-0-bbr-postgres-db.tar", tempDir, deployHelper.GetDeploymentName())
			files, err := filepath.Glob(tarBackupFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(files).NotTo(BeEmpty())

			By("Dropping the table")
			err = db.DropTable(pgprops.Databases.Databases[0].Name, "restore_0")
			Expect(err).NotTo(HaveOccurred())

			By("Restoring the database")
			cmd = exec.Command("bbr", "deployment", "--target", configParams.Bosh.Target, "--username", configParams.Bosh.Username, "--deployment", deployHelper.GetDeploymentName(), "restore", "--artifact-path", filepath.Dir(files[0]))
			err = cmd.Run()
			Expect(err).To(HaveOccurred())

			By("Validating that the dropped table has been restored")
			result, err = db.CheckTableExist("restore_0", pgprops.Databases.Databases[0].Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeTrue())
		})
	})
})
