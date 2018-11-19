package deploy_test

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Create a fresh deployment", func() {

	var sshKeyFile string
	var bosh_ssh_command string
	var pgprops helpers.Properties
	var pgHost string
	var db helpers.PGData

	Context("With a good role", func() {

		BeforeEach(func() {
			var err error
			deployHelper.SetOpDefs(nil)
			err = deployHelper.Deploy()
			Expect(err).NotTo(HaveOccurred())

			sshKeyFile, err = deployHelper.WriteSSHKey()
			Expect(err).NotTo(HaveOccurred())
			bosh_ssh_command = "source /var/vcap/jobs/postgres/bin/pgconfig.sh; export PGPASSWORD='%s'; $PACKAGE_DIR/bin/psql -p 5524 -U %s postgres -c 'select now()'"

			pgprops, pgHost, err = deployHelper.GetPGPropsAndHost()
			Expect(err).NotTo(HaveOccurred())
			db, err = deployHelper.ConnectToPostgres(pgHost, pgprops)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			var err error
			db.CloseConnections()
			err = os.Remove(sshKeyFile)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Successfully validates database configuration", func() {
			var err error
			pgData, err := db.GetData()
			Expect(err).NotTo(HaveOccurred())
			validator := helpers.NewValidator(pgprops, pgData, db, latestPostgreSQLVersion)
			err = validator.ValidateAll()
			Expect(err).NotTo(HaveOccurred())
		})

		It("Successfully uses vcap local connections", func() {
			var err error
			var cmd *exec.Cmd

			cmd = exec.Command("ssh", "-i", sshKeyFile, "-o", "BatchMode=yes", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), fmt.Sprintf(bosh_ssh_command, "fake", "vcap"))
			stdout, stderr, err := helpers.RunCommand(cmd)
			Expect(err).NotTo(HaveOccurred(), "stderr was: '%v', stdout was: '%v'", stderr, stdout)
		})

		It("Fails to use non vcap local connections", func() {
			var err error
			var cmd *exec.Cmd

			cmd = exec.Command("ssh", "-i", sshKeyFile, fmt.Sprintf("%s@%s", deployHelper.GetVariable("testuser_name"), pgHost), "-o", "BatchMode=yes", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf(bosh_ssh_command, deployHelper.GetVariable("defuser_password"), deployHelper.GetVariable("defuser_name")))
			stdout, stderr, err := helpers.RunCommand(cmd)
			Expect(err).NotTo(HaveOccurred(), "stderr was: '%v', stdout was: '%v'", stderr, stdout)
		})

	})

	It("Fails to deploy with a bad role", func() {
		deployHelper.SetOpDefs(helpers.Define_add_bad_role())
		err := deployHelper.Deploy()
		Expect(err).To(HaveOccurred())
	})
})
