package deploy_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Deploy single instance with a fresh deployment", func() {

	BeforeEach(func() {
		var err error
		DB.CloseConnections()
		DB, err = deployHelper.ConnectToPostgres(pgHost, pgprops)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Testing connections", func() {

		It("Successfully uses local connections", func() {
			var err error
			var sshKeyFile string
			var bosh_ssh_command string
			var cmd *exec.Cmd

			// TEST THAT VCAP LOCAL CONNECTION IS  TRUSTED
			sshKeyFile, err = deployHelper.WriteSSHKey()
			Expect(err).NotTo(HaveOccurred())
			bosh_ssh_command = "export PGPASSWORD='%s'; /var/vcap/packages/postgres-9.6.8/bin/psql -p 5524 -U %s postgres -c 'select now()'"
			cmd = exec.Command("ssh", "-i", sshKeyFile, "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), fmt.Sprintf(bosh_ssh_command, "fake", "vcap"))
			err = cmd.Run()
			Expect(err).NotTo(HaveOccurred())
			// TEST THAT NON-VCAP LOCAL CONNECTIONS ARE NOT TRUSTED
			cmd = exec.Command("ssh", "-i", sshKeyFile, fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf(bosh_ssh_command, "fake", deployHelper.GetVariable("defuser_name")))
			err = cmd.Run()
			Expect(err).To(HaveOccurred())
			cmd = exec.Command("ssh", "-i", sshKeyFile, fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf(bosh_ssh_command, deployHelper.GetVariable("defuser_password"), deployHelper.GetVariable("defuser_name")))
			err = cmd.Run()
			Expect(err).NotTo(HaveOccurred())
			err = os.Remove(sshKeyFile)
			Expect(err).NotTo(HaveOccurred())

		})

		It("Fails to update with a bad role", func() {
			var err error
			deployHelper.SetOpDefs(helpers.Define_add_bad_role())
			err = deployHelper.Deploy()
			Expect(err).To(HaveOccurred())

		})

		It("Successfully tests SSL connections", func() {
			var err error
			var sshKeyFile string
			var bosh_ssh_command string
			var cmd *exec.Cmd
			By("Enabling SSL")
			deployHelper.SetOpDefs(helpers.Define_ssl_ops())
			err = deployHelper.Deploy()
			Expect(err).NotTo(HaveOccurred())

			By("Re-initializing a postgres client connection")
			DB.CloseConnections()
			DB, err = deployHelper.ConnectToPostgres(pgHost, pgprops)
			Expect(err).NotTo(HaveOccurred())

			goodCACerts := deployHelper.GetDeployment().GetVariable("postgres_cert")
			rootCertPath, err := helpers.WriteFile(goodCACerts.(map[interface{}]interface{})["ca"].(string))
			Expect(err).NotTo(HaveOccurred())
			badCAcerts := deployHelper.GetDeployment().GetVariable(deployHelper.GetVariable("certs_bad_ca").(string))
			badCaPath, err := helpers.WriteFile(badCAcerts.(map[interface{}]interface{})["certificate"].(string))
			Expect(err).NotTo(HaveOccurred())

			By("Checking non-secure connections")
			_, err = DB.GetPostgreSQLVersion()
			if err != nil {
				Expect(err.Error()).NotTo(HaveOccurred())
			}
			// TEST THAT VCAP LOCAL CONNECTION IS  TRUSTED
			sshKeyFile, err = deployHelper.WriteSSHKey()
			Expect(err).NotTo(HaveOccurred())
			bosh_ssh_command = "/var/vcap/packages/postgres-9.6.8/bin/psql -p 5524 -U %s postgres -c 'select now()'"
			cmd = exec.Command("ssh", "-i", sshKeyFile, "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), fmt.Sprintf(bosh_ssh_command, "vcap"))
			err = cmd.Run()

			// TEST THAT NON-VCAP NON-SECURE LOCAL CONNECTIONS ARE NOT TRUSTED
			Expect(err).NotTo(HaveOccurred())
			cmd = exec.Command("ssh", "-i", sshKeyFile, fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf(bosh_ssh_command, deployHelper.GetVariable("defuser_name")))
			err = cmd.Run()
			Expect(err).To(HaveOccurred())
			err = os.Remove(sshKeyFile)
			Expect(err).NotTo(HaveOccurred())

			By("Checking secure connections")
			err = DB.ChangeSSLMode("verify-full", badCaPath)
			Expect(err).NotTo(HaveOccurred())
			_, err = DB.GetPostgreSQLVersion()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("x509"))

			err = os.Remove(badCaPath)
			Expect(err).NotTo(HaveOccurred())

			err = DB.ChangeSSLMode("verify-ca", rootCertPath)
			Expect(err).NotTo(HaveOccurred())
			_, err = DB.GetPostgreSQLVersion()
			if err != nil {
				Expect(err.Error()).NotTo(HaveOccurred())
			}

			err = DB.ChangeSSLMode("verify-full", rootCertPath)
			Expect(err).NotTo(HaveOccurred())
			_, err = DB.GetPostgreSQLVersion()
			if err != nil {
				Expect(err.Error()).NotTo(HaveOccurred())
			}

			err = os.Remove(rootCertPath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Successfully uses cert authentication for client connection", func() {
			var err error
			var sshKeyFile string
			var bosh_ssh_command string
			var cmd *exec.Cmd
			deployHelper.SetOpDefs(helpers.Define_mutual_ssl_ops())
			err = deployHelper.Deploy()
			Expect(err).NotTo(HaveOccurred())

			By("Re-initializing a postgres client connection")
			DB.CloseConnections()
			DB, err = deployHelper.ConnectToPostgres(pgHost, pgprops)
			Expect(err).NotTo(HaveOccurred())

			certs := deployHelper.GetDeployment().GetVariable(deployHelper.GetVariable("certs_matching_certs").(string))
			goodCACerts := deployHelper.GetDeployment().GetVariable("postgres_cert")
			rootCertPath, err := helpers.WriteFile(goodCACerts.(map[interface{}]interface{})["ca"].(string))
			Expect(err).NotTo(HaveOccurred())
			err = DB.ChangeSSLMode("verify-full", rootCertPath)
			Expect(err).NotTo(HaveOccurred())
			err = DB.SetCertUserCertificates(deployHelper.GetVariable("certs_matching_name").(string), certs.(map[interface{}]interface{}))
			Expect(err).NotTo(HaveOccurred())
			err = DB.UseCertAuthentication(true)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				_, err = DB.GetPostgreSQLVersion()
				if err != nil {
					return err.Error()
				}
				return ""
			}, "30s", "5s").Should(BeEmpty())
			// TEST THAT NON-VCAP SECURE LOCAL CONNECTIONS ARE NOT TRUSTED
			sshKeyFile, err = deployHelper.WriteSSHKey()
			Expect(err).NotTo(HaveOccurred())
			cmd = exec.Command("ssh", "-i", sshKeyFile, fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf(bosh_ssh_command, deployHelper.GetVariable("certs_matching_name")))
			err = cmd.Run()
			Expect(err).To(HaveOccurred())
			err = os.Remove(sshKeyFile)
			Expect(err).NotTo(HaveOccurred())

			certs = deployHelper.GetDeployment().GetVariable(deployHelper.GetVariable("certs_wrong_certs").(string))
			err = DB.SetCertUserCertificates(DB.Data.CertUser.Name, certs.(map[interface{}]interface{}))
			Expect(err).NotTo(HaveOccurred())
			_, err = DB.GetPostgreSQLVersion()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("certificate authentication failed"))

			certs = deployHelper.GetDeployment().GetVariable(deployHelper.GetVariable("certs_mapped_certs").(string))
			err = DB.SetCertUserCertificates(deployHelper.GetVariable("certs_mapped_name").(string), certs.(map[interface{}]interface{}))
			Expect(err).NotTo(HaveOccurred())
			_, err = DB.GetPostgreSQLVersion()
			Expect(err).NotTo(HaveOccurred())

		})
	})

	Context("Testing hooks", func() {
		It("Successfully manage hooks", func() {
			var err error
			var sshKeyFile string
			var bosh_ssh_command string
			var cmd *exec.Cmd
			By("Deploying hooks")
			psql_command := "${PACKAGE_DIR}/bin/psql -U vcap -p ${PORT} -d postgres -c 'CREATE ROLE %s WITH LOGIN'"
			pre_start_uuid := helpers.GetUUID()
			post_start_role_name := "poststartuser"
			pre_stop_role_name := "prestopuser"
			post_stop_uuid := helpers.GetUUID()

			pre_start_value := fmt.Sprintf("echo %s", pre_start_uuid)
			post_start_value := fmt.Sprintf(psql_command, post_start_role_name)
			pre_stop_value := fmt.Sprintf(psql_command, pre_stop_role_name)
			post_stop_value := fmt.Sprintf("echo %s", post_stop_uuid)

			jan := helpers.Janitor{
				Script: `${PACKAGE_DIR}/bin/psql -U vcap -p ${PORT} -d postgres << EOF
CREATE TABLE IF NOT EXISTS test_hook(name VARCHAR(10) NOT NULL UNIQUE,total INTEGER NOT NULL);
INSERT INTO test_hook (name, total) VALUES ('test', 1) ON CONFLICT (name) DO NOTHING;
UPDATE test_hook SET total = total + 1 WHERE name = 'test';
EOF
`,
				Timeout:  60,
				Interval: 1,
			}

			deployHelper.SetOpDefs(append(jan.GetOpDefinitions(), helpers.DefineHooks("0", pre_start_value, post_start_value, pre_stop_value, post_stop_value)...))
			err = deployHelper.Deploy()
			Expect(err).NotTo(HaveOccurred())

			sshKeyFile, err = deployHelper.WriteSSHKey()
			Expect(err).NotTo(HaveOccurred())

			By("Testing the pre-start hook")
			bosh_ssh_command = "source /var/vcap/jobs/postgres/bin/pgconfig.sh; grep %s ${HOOK_LOG_OUT}"
			cmd = exec.Command("ssh", "-i", sshKeyFile, "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), fmt.Sprintf(bosh_ssh_command, pre_start_uuid))
			err = cmd.Run()
			Expect(err).NotTo(HaveOccurred())

			By("Testing the post-start hook")
			role_exist, err := DB.CheckRoleExist(post_start_role_name)
			Expect(err).NotTo(HaveOccurred())
			Expect(role_exist).To(BeTrue())

			By("Restarting postgres node")
			err = deployHelper.GetDeployment().Restart("postgres")
			Expect(err).NotTo(HaveOccurred())

			By("Testing the pre-stop hook")
			role_exist, err = DB.CheckRoleExist(pre_stop_role_name)
			Expect(err).NotTo(HaveOccurred())
			Expect(role_exist).To(BeTrue())

			By("Testing the post-stop hook")
			bosh_ssh_command = "source /var/vcap/jobs/postgres/bin/pgconfig.sh; grep %s ${HOOK_LOG_OUT}"
			cmd = exec.Command("ssh", "-i", sshKeyFile, "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), fmt.Sprintf(bosh_ssh_command, post_stop_uuid))
			err = cmd.Run()
			Expect(err).NotTo(HaveOccurred())
			err = os.Remove(sshKeyFile)

			By("Testing the frequency-based hook")
			conn, err := DB.GetSuperUserConnection()
			Eventually(func() int {
				rows, err := conn.Run("select total from test_hook where name = 'test'")
				var counter struct {
					Total int `json:"total"`
				}
				Expect(err).NotTo(HaveOccurred())
				err = json.Unmarshal([]byte(rows[0]), &counter)
				Expect(err).NotTo(HaveOccurred())
				return counter.Total
			}, "15s", "2s").Should(BeNumerically(">", 10))
		})

		It("Successfully restarts janitor in case of failure", func() {
			var err error
			var sshKeyFile string
			var bosh_ssh_command string
			var cmd *exec.Cmd
			jan := helpers.Janitor{
				Script: `#!/bin/bash
STATEFILE=/tmp/statefile
if [ -f $STATEFILE ]; then
  echo second start >> $STATEFILE
else
  touch $STATEFILE
  chmod 777 $STATEFILE
  exit 1
fi`,
				Timeout:  60,
				Interval: 86400,
			}
			deployHelper.SetOpDefs(jan.GetOpDefinitions())
			err = deployHelper.Deploy()
			Expect(err).NotTo(HaveOccurred())

			sshKeyFile, err = deployHelper.WriteSSHKey()
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				bosh_ssh_command = "grep second /tmp/statefile"
				cmd = exec.Command("ssh", "-i", sshKeyFile, "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), bosh_ssh_command)
				err = cmd.Run()
				if err != nil {
					return err.Error()
				}
				return ""
			}, "10s", "2s").Should(BeEmpty())
			err = os.Remove(sshKeyFile)
		})

		It("Successfully starts postgres in case of a hook fails", func() {
			var err error
			var sshKeyFile string
			var bosh_ssh_command string
			var cmd *exec.Cmd
			pre_start_uuid := helpers.GetUUID()
			deployHelper.SetOpDefs(helpers.DefineHooks("3", fmt.Sprintf("for i in $(seq 10); do echo %s-$i; sleep 1; done", pre_start_uuid), "", "", ""))
			err = deployHelper.Deploy()
			Expect(err).NotTo(HaveOccurred())

			sshKeyFile, err = deployHelper.WriteSSHKey()
			Expect(err).NotTo(HaveOccurred())
			bosh_ssh_command = "source /var/vcap/jobs/postgres/bin/pgconfig.sh; if ! grep %s-10 ${HOOK_LOG_OUT}; then exit 0; else exit 1; fi"
			cmd = exec.Command("ssh", "-i", sshKeyFile, "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), fmt.Sprintf(bosh_ssh_command, pre_start_uuid))
			err = cmd.Run()
			Expect(err).NotTo(HaveOccurred())
			_, err = DB.GetPostgreSQLVersion()
			Expect(err).NotTo(HaveOccurred())
			err = os.Remove(sshKeyFile)
		})

	})

})
