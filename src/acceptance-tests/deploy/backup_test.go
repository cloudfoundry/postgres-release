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

		caCertPath, err = helpers.WriteFile(configParams.Bosh.Credentials.CACert)
		Expect(err).NotTo(HaveOccurred())

		tempDir, err = helpers.CreateTempDir()
		Expect(err).NotTo(HaveOccurred())
		pwd, err = os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		err = os.Chdir(tempDir)
		Expect(err).NotTo(HaveOccurred())

		os.Setenv("BOSH_CLIENT_SECRET", configParams.Bosh.Credentials.ClientSecret)
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

		It("Fails to backup the database", func() {
			var err error
			var cmd *exec.Cmd
			cmd = exec.Command("bbr", "deployment", "--target", configParams.Bosh.Target, "--username", configParams.Bosh.Credentials.Client, "--deployment", deployHelper.GetDeploymentName(), "backup")
			err = cmd.Run()
			Expect(err).To(HaveOccurred())
		})

		It("Fails to restore the database", func() {
			var err error
			var cmd *exec.Cmd
			cmd = exec.Command("bbr", "deployment", "--target", configParams.Bosh.Target, "--username", configParams.Bosh.Credentials.Client, "--deployment", deployHelper.GetDeploymentName(), "restore", "--artifact-path", fmt.Sprintf("%s/doesnotexist", tempDir))
			err = cmd.Run()
			Expect(err).To(HaveOccurred())
		})

	})

	Context("BBR is enabled", func() {

		var pgprops helpers.Properties
		var pgHost string
		var db helpers.PGData
		JustBeforeEach(func() {
			var err error

			pgprops, pgHost, err = deployHelper.GetPGPropsAndHost()
			Expect(err).NotTo(HaveOccurred())
			db, err = deployHelper.ConnectToPostgres(pgHost, pgprops)
			Expect(err).NotTo(HaveOccurred())
		})

		AssertBackupRestoreSuccessful := func() func() {
			return func() {
				var err error
				var cmd *exec.Cmd
				By("Running pre-backup-checks")
				cmd = exec.Command("bbr", "deployment", "--target", configParams.Bosh.Target, "--username", configParams.Bosh.Credentials.Client, "--deployment", deployHelper.GetDeploymentName(), "pre-backup-check")
				stdout, stderr, err := helpers.RunCommand(cmd)
				Expect(err).NotTo(HaveOccurred(), "stderr was: '%v', stdout was: '%v'", stderr, stdout)

				By("Changing content")
				err = db.CreateAndPopulateTablesWithPrefix(pgprops.Databases.Databases[0].Name, helpers.Test1Load, "restore")
				Expect(err).NotTo(HaveOccurred())
				result, err := db.CheckTableExist("restore_0", pgprops.Databases.Databases[0].Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(BeTrue())

				By("Running backup")
				cmd = exec.Command("bbr", "deployment", "--target", configParams.Bosh.Target, "--username", configParams.Bosh.Credentials.Client, "--deployment", deployHelper.GetDeploymentName(), "backup")
				stdout, stderr, err = helpers.RunCommand(cmd)
				Expect(err).NotTo(HaveOccurred(), "stderr was: '%v', stdout was: '%v'", stderr, stdout)
				tarBackupFile := fmt.Sprintf("%s/%s*/*-bbr-postgres-db.tar", tempDir, deployHelper.GetDeploymentName())
				files, err := filepath.Glob(tarBackupFile)
				Expect(err).NotTo(HaveOccurred())
				Expect(files).NotTo(BeEmpty())

				By("Dropping the table")
				err = db.DropTable(pgprops.Databases.Databases[0].Name, "restore_0")
				Expect(err).NotTo(HaveOccurred())

				By("Restoring the database")
				cmd = exec.Command("bbr", "deployment", "--target", configParams.Bosh.Target, "--username", configParams.Bosh.Credentials.Client, "--deployment", deployHelper.GetDeploymentName(), "restore", "--artifact-path", filepath.Dir(files[0]))
				stdout, stderr, err = helpers.RunCommand(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("Validating that the dropped table has been restored")
				result, err = db.CheckTableExist("restore_0", pgprops.Databases.Databases[0].Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(BeTrue())

				By("Dropping the table")
				err = db.DropTable(pgprops.Databases.Databases[0].Name, "restore_0")
				Expect(err).NotTo(HaveOccurred())
			}
		}

		Context("BBR job is colocated", func() {
			Context("When using BOSH links", func() {
				BeforeEach(func() {
					deployHelper.SetOpDefs(helpers.Define_bbr_ops())
				})

				It("Fails to restore the database", func() {
					var err error
					var cmd *exec.Cmd
					cmd = exec.Command("bbr", "deployment", "--target", configParams.Bosh.Target, "--username", configParams.Bosh.Credentials.Client, "--deployment", deployHelper.GetDeploymentName(), "restore", "--artifact-path", fmt.Sprintf("%s/doesnotexist", tempDir))
					err = cmd.Run()
					Expect(err).To(HaveOccurred())
				})

				It("Successfully backup and restore the database", AssertBackupRestoreSuccessful())
			})

			Context("When not using BOSH links", func() {
				BeforeEach(func() {
					deployHelper.SetOpDefs(helpers.Define_bbr_no_link_ops())
				})

				It("Successfully backup and restore the database", AssertBackupRestoreSuccessful())
			})
		})

		Context("BBR job is not colocated", func() {
			Context("With SSL disabled", func() {
				BeforeEach(func() {
					deployHelper.SetOpDefs(helpers.Define_bbr_not_colocated_ops())
				})

				It("Successfully backup and restore the database", AssertBackupRestoreSuccessful())
			})

			Context("With SSL", func() {
				BeforeEach(func() {
					deployHelper.SetOpDefs(helpers.Define_bbr_ssl_verify_ca())
				})

				It("Successfully backup and restore the database", AssertBackupRestoreSuccessful())
			})

			Context("With SSL enforced with hostname validation", func() {
				BeforeEach(func() {
					deployHelper.SetOpDefs(helpers.Define_bbr_ssl_verify_full())
				})

				It("Successfully backup and restore the database", AssertBackupRestoreSuccessful())
			})

			Context("With SSL authentication", func() {
				BeforeEach(func() {
					deployHelper.SetOpDefs(helpers.Define_bbr_client_certs())
				})

				It("Successfully backup and restore the database", AssertBackupRestoreSuccessful())
			})
		})
	})
})
