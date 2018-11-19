package deploy_test

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SSL enabled", func() {
	var sshKeyFile string
	var bosh_ssh_command string
	var pgHost string
	var db helpers.PGData

	JustBeforeEach(func() {
		var pgprops helpers.Properties
		var err error

		err = deployHelper.Deploy()
		Expect(err).NotTo(HaveOccurred())

		pgprops, pgHost, err = deployHelper.GetPGPropsAndHost()
		Expect(err).NotTo(HaveOccurred())
		db, err = deployHelper.ConnectToPostgres(pgHost, pgprops)
		Expect(err).NotTo(HaveOccurred())

		sshKeyFile, err = deployHelper.WriteSSHKey()
		Expect(err).NotTo(HaveOccurred())

		bosh_ssh_command = "source /var/vcap/jobs/postgres/bin/pgconfig.sh; $PACKAGE_DIR/bin/psql -p 5524 -U %s postgres -c 'select now()'"
	})

	AfterEach(func() {
		var err error
		err = os.Remove(sshKeyFile)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("SSL connection enabled", func() {

		BeforeEach(func() {
			deployHelper.SetOpDefs(helpers.Define_ssl_ops())
		})

		It("Successfully trust vcap local connections", func() {
			var cmd *exec.Cmd
			var err error
			cmd = exec.Command("ssh", "-i", sshKeyFile, "-o", "BatchMode=yes", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), fmt.Sprintf(bosh_ssh_command, "vcap"))
			stdout, stderr, err := helpers.RunCommand(cmd)
			Expect(err).NotTo(HaveOccurred(), "stderr was: '%v', stdout was: '%v'", stderr, stdout)
		})

		It("Fails to trust non-vcap local connections", func() {
			var cmd *exec.Cmd
			var err error
			cmd = exec.Command("ssh", "-i", sshKeyFile, fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), "-o", "BatchMode=yes", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf(bosh_ssh_command, deployHelper.GetVariable("defuser_name")))
			err = cmd.Run()
			Expect(err).To(HaveOccurred())
		})

		It("Successfully connect using good certificates", func() {
			var err error

			goodCACerts := deployHelper.GetDeployment().GetVariable("postgres_cert")
			Expect(err).NotTo(HaveOccurred())

			err = db.ChangeSSLMode("verify-ca", goodCACerts.(map[interface{}]interface{})["ca"].(string))
			Expect(err).NotTo(HaveOccurred())
			_, err = db.GetPostgreSQLVersion()
			if err != nil {
				Expect(err.Error()).NotTo(HaveOccurred())
			}

			err = db.ChangeSSLMode("verify-full", goodCACerts.(map[interface{}]interface{})["ca"].(string))
			Expect(err).NotTo(HaveOccurred())
			_, err = db.GetPostgreSQLVersion()
			if err != nil {
				Expect(err.Error()).NotTo(HaveOccurred())
			}
		})

		It("Fails to connect using bad certificates", func() {
			var err error

			badCAcerts := deployHelper.GetDeployment().GetVariable(deployHelper.GetVariable("certs_bad_ca").(string))
			Expect(err).NotTo(HaveOccurred())

			err = db.ChangeSSLMode("verify-full", badCAcerts.(map[interface{}]interface{})["certificate"].(string))
			Expect(err).NotTo(HaveOccurred())
			_, err = db.GetPostgreSQLVersion()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("x509"))
		})
	})

	Describe("Mutual certificate authentication", func() {

		BeforeEach(func() {
			deployHelper.SetOpDefs(helpers.Define_mutual_ssl_ops())
		})

		It("Fails to trust secure non-vcap local connections", func() {
			var err error
			var cmd *exec.Cmd
			cmd = exec.Command("ssh", "-i", sshKeyFile, fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), "-o", "BatchMode=yes", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf(bosh_ssh_command, deployHelper.GetVariable("certs_matching_name")))
			err = cmd.Run()
			Expect(err).To(HaveOccurred())
		})

		Context("Testing non-local connections", func() {

			JustBeforeEach(func() {
				var err error
				goodCACerts := deployHelper.GetDeployment().GetVariable("postgres_cert")
				Expect(err).NotTo(HaveOccurred())
				err = db.ChangeSSLMode("verify-full", goodCACerts.(map[interface{}]interface{})["ca"].(string))
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				var err error
				if db.Data.SSLRootCert != "" {
					err = os.Remove(db.Data.SSLRootCert)
					Expect(err).NotTo(HaveOccurred())
				}
				if db.Data.CertUser.Certificate != "" {
					err = os.Remove(db.Data.CertUser.Certificate)
					Expect(err).NotTo(HaveOccurred())
				}
				if db.Data.CertUser.Key != "" {
					err = os.Remove(db.Data.CertUser.Key)
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("Successfully authenticate remote user using good certificate", func() {
				var err error

				certs := deployHelper.GetDeployment().GetVariable(deployHelper.GetVariable("certs_matching_certs").(string))
				err = db.SetCertUserCertificates(deployHelper.GetVariable("certs_matching_name").(string), certs.(map[interface{}]interface{}))
				Expect(err).NotTo(HaveOccurred())
				err = db.UseCertAuthentication(true)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() string {
					_, err = db.GetPostgreSQLVersion()
					if err != nil {
						return err.Error()
					}
					return ""
				}, "30s", "5s").Should(BeEmpty())
			})

			It("Fails to authenticate remote user using bad certitifcates", func() {
				var err error
				certs := deployHelper.GetDeployment().GetVariable(deployHelper.GetVariable("certs_wrong_certs").(string))
				err = db.SetCertUserCertificates(deployHelper.GetVariable("certs_matching_name").(string), certs.(map[interface{}]interface{}))
				Expect(err).NotTo(HaveOccurred())
				err = db.UseCertAuthentication(true)
				Expect(err).NotTo(HaveOccurred())
				_, err = db.GetPostgreSQLVersion()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("certificate authentication failed"))
			})

			It("Successfully authenticates remote user using good certificates with mapped common name", func() {
				var err error
				certs := deployHelper.GetDeployment().GetVariable(deployHelper.GetVariable("certs_mapped_certs").(string))
				err = db.SetCertUserCertificates(deployHelper.GetVariable("certs_mapped_name").(string), certs.(map[interface{}]interface{}))
				Expect(err).NotTo(HaveOccurred())
				err = db.UseCertAuthentication(true)
				Expect(err).NotTo(HaveOccurred())
				_, err = db.GetPostgreSQLVersion()
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
