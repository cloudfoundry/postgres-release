package deploy_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test hooks", func() {

	var sshKeyFile string
	var pgHost string
	var db helpers.PGData

	BeforeEach(func() {
		var pgprops helpers.Properties
		var err error
		pgprops, pgHost, err = deployHelper.GetPGPropsAndHost()
		Expect(err).NotTo(HaveOccurred())
		db, err = deployHelper.ConnectToPostgres(pgHost, pgprops)
		Expect(err).NotTo(HaveOccurred())
	})

	JustBeforeEach(func() {
		var err error
		err = deployHelper.Deploy()
		Expect(err).NotTo(HaveOccurred())

		sshKeyFile, err = deployHelper.WriteSSHKey()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := os.Remove(sshKeyFile)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Postgres hooks run successfully", func() {
		var pre_start_uuid string
		var post_start_role_name string
		var pre_stop_role_name string
		var post_stop_uuid string

		BeforeEach(func() {
			psql_command := "${PACKAGE_DIR}/bin/psql -U vcap -p ${PORT} -d postgres -c 'CREATE ROLE %s WITH LOGIN'"
			pre_start_uuid = helpers.GetUUID()
			post_start_role_name = "poststartuser"
			pre_stop_role_name = "prestopuser"
			post_stop_uuid = helpers.GetUUID()

			pre_start_value := fmt.Sprintf("echo %s", pre_start_uuid)
			post_start_value := fmt.Sprintf(psql_command, post_start_role_name)
			pre_stop_value := fmt.Sprintf(psql_command, pre_stop_role_name)
			post_stop_value := fmt.Sprintf("echo %s", post_stop_uuid)

			deployHelper.SetOpDefs(helpers.DefineHooks("0", pre_start_value, post_start_value, pre_stop_value, post_stop_value))
		})

		It("Successfully manage hooks", func() {
			var err error
			var bosh_ssh_command string
			var cmd *exec.Cmd

			By("Testing the pre-start hook")
			bosh_ssh_command = "source /var/vcap/jobs/postgres/bin/pgconfig.sh; grep %s ${HOOK_LOG_OUT}"
			cmd = exec.Command("ssh", "-i", sshKeyFile, "-o", "BatchMode=yes", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), fmt.Sprintf(bosh_ssh_command, pre_start_uuid))
			stdout, stderr, err := helpers.RunCommand(cmd)
			Expect(err).NotTo(HaveOccurred(), "stderr was: '%v', stdout was: '%v'", stderr, stdout)

			By("Testing the post-start hook")
			role_exist, err := db.CheckRoleExist(post_start_role_name)
			Expect(err).NotTo(HaveOccurred())
			Expect(role_exist).To(BeTrue())

			By("Restarting postgres node")
			err = deployHelper.GetDeployment().Restart("postgres")
			Expect(err).NotTo(HaveOccurred())

			By("Testing the pre-stop hook")
			role_exist, err = db.CheckRoleExist(pre_stop_role_name)
			Expect(err).NotTo(HaveOccurred())
			Expect(role_exist).To(BeTrue())

			By("Testing the post-stop hook")
			bosh_ssh_command = "source /var/vcap/jobs/postgres/bin/pgconfig.sh; grep %s ${HOOK_LOG_OUT}"
			cmd = exec.Command("ssh", "-i", sshKeyFile, "-o", "BatchMode=yes", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), fmt.Sprintf(bosh_ssh_command, post_stop_uuid))
			stdout, stderr, err = helpers.RunCommand(cmd)
			Expect(err).NotTo(HaveOccurred(), "stderr was: '%v', stdout was: '%v'", stderr, stdout)
		})
	})

	Context("Postgres hooks fail to run", func() {

		var pre_start_uuid string

		BeforeEach(func() {
			pre_start_uuid := helpers.GetUUID()
			deployHelper.SetOpDefs(helpers.DefineHooks("3", fmt.Sprintf("for i in $(seq 10); do echo %s-$i; sleep 1; done", pre_start_uuid), "", "", ""))
		})

		It("Successfully starts postgres", func() {
			var err error
			var bosh_ssh_command string
			var cmd *exec.Cmd

			bosh_ssh_command = "source /var/vcap/jobs/postgres/bin/pgconfig.sh; if ! grep %s-10 ${HOOK_LOG_OUT}; then exit 0; else exit 1; fi"
			cmd = exec.Command("ssh", "-i", sshKeyFile, "-o", "BatchMode=yes", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), fmt.Sprintf(bosh_ssh_command, pre_start_uuid))
			stdout, stderr, err := helpers.RunCommand(cmd)
			Expect(err).NotTo(HaveOccurred(), "stderr was: '%v', stdout was: '%v'", stderr, stdout)
			_, err = db.GetPostgreSQLVersion()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Janitor script runs successfully", func() {
		BeforeEach(func() {
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

			deployHelper.SetOpDefs(jan.GetOpDefinitions())
		})

		It("Runs the script with the expected frequency", func() {
			conn, err := db.GetSuperUserConnection()
			Expect(err).NotTo(HaveOccurred())
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
	})

	Context("Janitor job starts correctly", func() {
		BeforeEach(func() {
			deployHelper.SetOpDefs(nil)
		})
		It("Successfully stops janitor", func() {
			var err error
			var bosh_ssh_command string
			var cmd *exec.Cmd

			By("Stopping the postgres node")
			err = deployHelper.GetDeployment().Stop("postgres")
			Expect(err).NotTo(HaveOccurred())

			By("Checking that janitor childs are stopped")
			// We expected two processes to exist because of our ssh command:
			// sshuser    10787   10786  bash -c ps -ef | grep janitor
			// sshuser    10789   10787  grep janitor
			bosh_ssh_command = "ps -ef | grep -c janitor"
			cmd = exec.Command("ssh", "-i", sshKeyFile, "-o", "BatchMode=yes", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), bosh_ssh_command)
			stdout, stderr, err := helpers.RunCommand(cmd)
			Expect(err).NotTo(HaveOccurred(), "stderr was: '%v', stdout was: '%v'", stderr, stdout)
			Expect(strings.Trim(stdout, " \n\t\r")).To(Equal("2"))

			By("Restarting the stopped postgres node")
			err = deployHelper.GetDeployment().Start("postgres")
			Expect(err).NotTo(HaveOccurred())

		})
	})

	Context("Janitor script fails to run", func() {

		BeforeEach(func() {
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
		})

		It("Successfully restarts janitor", func() {
			var err error
			var bosh_ssh_command string
			var cmd *exec.Cmd

			Eventually(func() string {
				bosh_ssh_command = "grep second /tmp/statefile"
				cmd = exec.Command("ssh", "-i", sshKeyFile, "-o", "BatchMode=yes", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), bosh_ssh_command)
				err = cmd.Run()
				if err != nil {
					return err.Error()
				}
				return ""
			}, "10s", "2s").Should(BeEmpty())
		})
	})
})
