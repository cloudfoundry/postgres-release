package deploy_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func createOrUpdateDeployment(postgresReleaseVersion int, manifestPath string, name string, variables map[string]interface{}, opDefs []helpers.OpDefinition) error {
	var err error
	var vars map[string]interface{}
	if variables != nil {
		vars = variables
	} else {
		vars = make(map[string]interface{})
	}
	releases := make(map[string]string)
	if postgresReleaseVersion != -1 {
		releases["postgres"] = strconv.Itoa(postgresReleaseVersion)
		err = director.UploadPostgresReleaseFromURL(postgresReleaseVersion)
		if err != nil {
			return err
		}
	}
	err = director.SetDeploymentFromManifest(manifestPath, releases, name)
	if err != nil {
		return err
	}
	if director.GetEnv(name).ContainsVariables() || variables != nil {
		if director.GetEnv(name).ContainsVariables() {
			if _, err = director.GetEnv(name).GetVmAddress("postgres"); err != nil {

				vars["postgres_host"] = "1.1.1.1"
				err = director.GetEnv(name).EvaluateTemplate(vars, opDefs, helpers.EvaluateOptions{})
				if err != nil {
					return err
				}
				err = director.GetEnv(name).CreateOrUpdateDeployment()
				if err != nil {
					return err
				}
			}
			var pgHost string
			pgHost, err = director.GetEnv(name).GetVmDNS("postgres")
			if err != nil {
				pgHost, err = director.GetEnv(name).GetVmAddress("postgres")
				if err != nil {
					return err
				}
			}
			vars["postgres_host"] = pgHost

			err = director.SetDeploymentFromManifest(manifestPath, releases, name)
			if err != nil {
				return err
			}
		}
		err = director.GetEnv(name).EvaluateTemplate(vars, opDefs, helpers.EvaluateOptions{})
		if err != nil {
			return err
		}
	}
	err = director.GetEnv(name).CreateOrUpdateDeployment()
	if err != nil {
		return err
	}
	return nil
}
func getPostgresJobProps(envName string) (helpers.Properties, error) {
	var err error
	manifestProps, err := director.GetEnv(envName).GetJobsProperties()
	if err != nil {
		return helpers.Properties{}, err
	}
	pgprops := manifestProps.GetJobProperties("postgres")[0]
	return pgprops, nil
}

func getPGPropsAndHost(envName string) (helpers.Properties, string, error) {

	pgprops, err := getPostgresJobProps(envName)
	if err != nil {
		return helpers.Properties{}, "", err
	}
	var pgHost string
	pgHost, err = director.GetEnv(envName).GetVmDNS("postgres")
	if err != nil {
		pgHost, err = director.GetEnv(envName).GetVmAddress("postgres")
		if err != nil {
			return pgprops, "", err
		}
	}
	return pgprops, pgHost, nil
}
func connectToPostgres(pgHost string, pgprops helpers.Properties, variables map[string]interface{}) (helpers.PGData, error) {

	pgc := helpers.PGCommon{
		Address: pgHost,
		Port:    pgprops.Databases.Port,
		DefUser: helpers.User{
			Name:     variables["defuser_name"].(string),
			Password: variables["defuser_password"].(string),
		},
		AdminUser: helpers.User{
			Name:     variables["superuser_name"].(string),
			Password: variables["superuser_password"].(string),
		},
		CertUser: helpers.User{},
	}
	DB, err := helpers.NewPostgres(pgc)
	if err != nil {
		return helpers.PGData{}, err
	}
	return DB, nil
}

func writeSSHKey(envName string) (string, error) {
	sshKey := director.GetEnv(envName).GetVariable("sshkey")
	keyPath, err := helpers.WriteFile(sshKey.(map[interface{}]interface{})["private_key"].(string))
	if err != nil {
		// set permission to 600
		err = helpers.SetPermissions(keyPath, 0600)
	}
	return keyPath, err
}

var _ = Describe("Deploy single instance", func() {

	var envName string
	var DB helpers.PGData
	var pgprops helpers.Properties
	var manifestPath, deploymentPrefix string
	var version int
	var latestPostgreSQLVersion string
	var variables map[string]interface{}
	var pgHost string

	JustBeforeEach(func() {
		manifestPath = "../testing/templates/postgres_simple.yml"
		var err error
		envName = helpers.GenerateEnvName(deploymentPrefix)
		latestPostgreSQLVersion = configParams.PostgreSQLVersion
		if latestPostgreSQLVersion == "current" {
			latestPostgreSQLVersion = versions.GetPostgreSQLVersion(versions.GetLatestVersion())
		}

		variables["superuser_name"] = "superuser"
		variables["superuser_password"] = "superpsw"
		variables["testuser_name"] = "sshuser"

		By("Deploying a single postgres instance")
		err = createOrUpdateDeployment(version, manifestPath, envName, variables, nil)
		Expect(err).NotTo(HaveOccurred())

		By("Initializing a postgres client connection")
		pgprops, pgHost, err = getPGPropsAndHost(envName)
		Expect(err).NotTo(HaveOccurred())
		DB, err = connectToPostgres(pgHost, pgprops, variables)
		Expect(err).NotTo(HaveOccurred())
		By("Populating the database")
		err = DB.CreateAndPopulateTables(pgprops.Databases.Databases[0].Name, helpers.SmallLoad)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("With a fresh deployment", func() {

		BeforeEach(func() {
			version = -1
			deploymentPrefix = "fresh"
			variables = make(map[string]interface{})
			variables["defuser_name"] = "pgadmin"
			variables["defuser_password"] = "adm$in!"

			variables["certs_matching_certs"] = "certuser_matching_certs"
			variables["certs_matching_name"] = "certuser_matching_name"

			variables["certs_mapped_certs"] = "certuser_mapped_certs"
			variables["certs_mapped_name"] = "certuser_mapped_name"
			variables["certs_mapped_cn"] = "certuser mapped cn"

			variables["certs_wrong_certs"] = "certuser_wrong_certs"
			variables["certs_wrong_cn"] = "certuser_wrong_cn"

			variables["certs_bad_ca"] = "bad_ca"
		})

		It("Successfully manage hooks", func() {
			var err error
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

			err = createOrUpdateDeployment(version, manifestPath, envName, variables, append(jan.GetOpDefinitions(), helpers.DefineHooks("0", pre_start_value, post_start_value, pre_stop_value, post_stop_value)...))
			Expect(err).NotTo(HaveOccurred())

			sshKeyFile, err := writeSSHKey(envName)
			Expect(err).NotTo(HaveOccurred())

			By("Testing the pre-start hook")
			bosh_ssh_command := "source /var/vcap/jobs/postgres/bin/pgconfig.sh; grep %s ${HOOK_LOG_OUT}"
			cmd := exec.Command("ssh", "-i", sshKeyFile, "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", variables["testuser_name"], pgHost), fmt.Sprintf(bosh_ssh_command, pre_start_uuid))
			err = cmd.Run()
			Expect(err).NotTo(HaveOccurred())

			By("Testing the post-start hook")
			role_exist, err := DB.CheckRoleExist(post_start_role_name)
			Expect(err).NotTo(HaveOccurred())
			Expect(role_exist).To(BeTrue())

			By("Restarting postgres node")
			err = director.GetEnv(envName).Restart("postgres")
			Expect(err).NotTo(HaveOccurred())

			By("Testing the pre-stop hook")
			role_exist, err = DB.CheckRoleExist(pre_stop_role_name)
			Expect(err).NotTo(HaveOccurred())
			Expect(role_exist).To(BeTrue())

			By("Testing the post-stop hook")
			bosh_ssh_command = "source /var/vcap/jobs/postgres/bin/pgconfig.sh; grep %s ${HOOK_LOG_OUT}"
			cmd = exec.Command("ssh", "-i", sshKeyFile, "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", variables["testuser_name"], pgHost), fmt.Sprintf(bosh_ssh_command, post_stop_uuid))
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

			By("Verifying that janitor script failure causes monit to restart janitor")
			jan = helpers.Janitor{
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

			err = createOrUpdateDeployment(version, manifestPath, envName, variables, jan.GetOpDefinitions())
			Expect(err).NotTo(HaveOccurred())

			sshKeyFile, err = writeSSHKey(envName)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				bosh_ssh_command = "grep second /tmp/statefile"
				cmd = exec.Command("ssh", "-i", sshKeyFile, "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", variables["testuser_name"], pgHost), bosh_ssh_command)
				err = cmd.Run()
				if err != nil {
					return err.Error()
				}
				return ""
			}, "10s", "2s").Should(BeEmpty())
			err = os.Remove(sshKeyFile)

			By("Verifying that hooks failure does not prevent postgres to start")
			pre_start_uuid = helpers.GetUUID()
			err = createOrUpdateDeployment(version, manifestPath, envName, variables, helpers.DefineHooks("3", fmt.Sprintf("for i in $(seq 10); do echo %s-$i; sleep 1; done", pre_start_uuid), "", "", ""))
			Expect(err).NotTo(HaveOccurred())
			sshKeyFile, err = writeSSHKey(envName)
			Expect(err).NotTo(HaveOccurred())
			bosh_ssh_command = "source /var/vcap/jobs/postgres/bin/pgconfig.sh; if ! grep %s-10 ${HOOK_LOG_OUT}; then exit 0; else exit 1; fi"
			cmd = exec.Command("ssh", "-i", sshKeyFile, "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", variables["testuser_name"], pgHost), fmt.Sprintf(bosh_ssh_command, pre_start_uuid))
			err = cmd.Run()
			Expect(err).NotTo(HaveOccurred())
			_, err = DB.GetPostgreSQLVersion()
			Expect(err).NotTo(HaveOccurred())
			err = os.Remove(sshKeyFile)
		})

		It("Successfully deploys a fresh env", func() {
			pgData, err := DB.GetData()
			Expect(err).NotTo(HaveOccurred())
			validator := helpers.NewValidator(pgprops, pgData, DB, latestPostgreSQLVersion)
			err = validator.ValidateAll()
			Expect(err).NotTo(HaveOccurred())

			By("Testing local connections")
			// TEST THAT VCAP LOCAL CONNECTION IS  TRUSTED
			sshKeyFile, err := writeSSHKey(envName)
			Expect(err).NotTo(HaveOccurred())
			bosh_ssh_command := "export PGPASSWORD='%s'; /var/vcap/packages/postgres-10.3/bin/psql -p 5524 -U %s postgres -c 'select now()'"
			cmd := exec.Command("ssh", "-i", sshKeyFile, "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", variables["testuser_name"], pgHost), fmt.Sprintf(bosh_ssh_command, "fake", "vcap"))
			err = cmd.Run()
			Expect(err).NotTo(HaveOccurred())
			// TEST THAT NON-VCAP LOCAL CONNECTIONS ARE NOT TRUSTED
			cmd = exec.Command("ssh", "-i", sshKeyFile, fmt.Sprintf("%s@%s", variables["testuser_name"], pgHost), "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf(bosh_ssh_command, "fake", variables["defuser_name"]))
			err = cmd.Run()
			Expect(err).To(HaveOccurred())
			cmd = exec.Command("ssh", "-i", sshKeyFile, fmt.Sprintf("%s@%s", variables["testuser_name"], pgHost), "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf(bosh_ssh_command, variables["defuser_password"], variables["defuser_name"]))
			err = cmd.Run()
			Expect(err).NotTo(HaveOccurred())
			err = os.Remove(sshKeyFile)
			Expect(err).NotTo(HaveOccurred())

			By("Updating with bad role")
			err = createOrUpdateDeployment(version, manifestPath, envName, variables, helpers.Define_add_bad_role())
			Expect(err).To(HaveOccurred())

			By("Enabling SSL")
			err = createOrUpdateDeployment(version, manifestPath, envName, variables, helpers.Define_ssl_ops())
			Expect(err).NotTo(HaveOccurred())
			By("Re-initializing a postgres client connection")
			DB.CloseConnections()
			DB, err = connectToPostgres(pgHost, pgprops, variables)
			Expect(err).NotTo(HaveOccurred())

			goodCACerts := director.GetEnv(envName).GetVariable("postgres_cert")
			rootCertPath, err := helpers.WriteFile(goodCACerts.(map[interface{}]interface{})["ca"].(string))
			Expect(err).NotTo(HaveOccurred())
			badCAcerts := director.GetEnv(envName).GetVariable(variables["certs_bad_ca"].(string))
			badCaPath, err := helpers.WriteFile(badCAcerts.(map[interface{}]interface{})["certificate"].(string))
			Expect(err).NotTo(HaveOccurred())

			By("Checking non-secure connections")
			_, err = DB.GetPostgreSQLVersion()
			if err != nil {
				Expect(err.Error()).NotTo(HaveOccurred())
			}
			// TEST THAT VCAP LOCAL CONNECTION IS  TRUSTED
			sshKeyFile, err = writeSSHKey(envName)
			Expect(err).NotTo(HaveOccurred())
			bosh_ssh_command = "/var/vcap/packages/postgres-10.3/bin/psql -p 5524 -U %s postgres -c 'select now()'"
			cmd = exec.Command("ssh", "-i", sshKeyFile, "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", variables["testuser_name"], pgHost), fmt.Sprintf(bosh_ssh_command, "vcap"))
			err = cmd.Run()

			// TEST THAT NON-VCAP NON-SECURE LOCAL CONNECTIONS ARE NOT TRUSTED
			Expect(err).NotTo(HaveOccurred())
			cmd = exec.Command("ssh", "-i", sshKeyFile, fmt.Sprintf("%s@%s", variables["testuser_name"], pgHost), "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf(bosh_ssh_command, variables["defuser_name"]))
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

			By("Using cert authentication for client connection")
			err = createOrUpdateDeployment(version, manifestPath, envName, variables, helpers.Define_mutual_ssl_ops())
			Expect(err).NotTo(HaveOccurred())
			By("Re-initializing a postgres client connection")
			DB.CloseConnections()
			DB, err = connectToPostgres(pgHost, pgprops, variables)
			Expect(err).NotTo(HaveOccurred())

			certs := director.GetEnv(envName).GetVariable(variables["certs_matching_certs"].(string))
			goodCACerts = director.GetEnv(envName).GetVariable("postgres_cert")
			rootCertPath, err = helpers.WriteFile(goodCACerts.(map[interface{}]interface{})["ca"].(string))
			Expect(err).NotTo(HaveOccurred())
			err = DB.ChangeSSLMode("verify-full", rootCertPath)
			Expect(err).NotTo(HaveOccurred())
			err = DB.SetCertUserCertificates(variables["certs_matching_name"].(string), certs.(map[interface{}]interface{}))
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
			sshKeyFile, err = writeSSHKey(envName)
			Expect(err).NotTo(HaveOccurred())
			cmd = exec.Command("ssh", "-i", sshKeyFile, fmt.Sprintf("%s@%s", variables["testuser_name"], pgHost), "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf(bosh_ssh_command, variables["certs_matching_name"]))
			err = cmd.Run()
			Expect(err).To(HaveOccurred())
			err = os.Remove(sshKeyFile)
			Expect(err).NotTo(HaveOccurred())

			certs = director.GetEnv(envName).GetVariable(variables["certs_wrong_certs"].(string))
			err = DB.SetCertUserCertificates(DB.Data.CertUser.Name, certs.(map[interface{}]interface{}))
			Expect(err).NotTo(HaveOccurred())
			_, err = DB.GetPostgreSQLVersion()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("certificate authentication failed"))

			certs = director.GetEnv(envName).GetVariable(variables["certs_mapped_certs"].(string))
			err = DB.SetCertUserCertificates(variables["certs_mapped_name"].(string), certs.(map[interface{}]interface{}))
			Expect(err).NotTo(HaveOccurred())
			_, err = DB.GetPostgreSQLVersion()
			Expect(err).NotTo(HaveOccurred())

		})
	})
	Describe("Upgrading an existent env", func() {

		var opDefs []helpers.OpDefinition

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
				err = createOrUpdateDeployment(-1, manifestPath, director.GetEnv(envName).Deployment.Name(), variables, opDefs)
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
				err = director.GetEnv(envName).Restart("postgres")
				Expect(err).NotTo(HaveOccurred())

				if deploymentPrefix == "upg-old-nocopy" {
					By("Validating the postgres-previous is not created")
					if !versions.IsMajor(latestPostgreSQLVersion, versions.GetOldVersion()) {
						sshKeyFile, err := writeSSHKey(envName)
						Expect(err).NotTo(HaveOccurred())
						cmd := exec.Command("ssh", "-i", sshKeyFile, "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", variables["testuser_name"], pgHost), "sudo test -d /var/vcap/store/postgres/postgres-previous")
						err = cmd.Run()
						Expect(err).To(HaveOccurred())
						err = os.Remove(sshKeyFile)
						Expect(err).NotTo(HaveOccurred())
					}
				} else if deploymentPrefix == "upg-old" {
					By("Validating the postgres-previous is created")
					sshKeyFile, err := writeSSHKey(envName)
					Expect(err).NotTo(HaveOccurred())
					cmd := exec.Command("ssh", "-i", sshKeyFile, "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", variables["testuser_name"], pgHost), "sudo test -d /var/vcap/store/postgres/postgres-previous")
					err = cmd.Run()
					Expect(err).NotTo(HaveOccurred())
					err = os.Remove(sshKeyFile)
					Expect(err).NotTo(HaveOccurred())
				}

			}
		}

		Context("Upgrading from older version", func() {
			BeforeEach(func() {
				version = versions.GetOlderVersion()
				deploymentPrefix = "upg-older"
				variables = make(map[string]interface{})
				variables["defuser_name"] = "pgadmin"
				variables["defuser_password"] = "admin"
				opDefs = nil
			})
			It("Successfully upgrades from older", AssertUpgradeSuccessful())
		})
		Context("Upgrading from old version", func() {
			BeforeEach(func() {
				version = versions.GetOldVersion()
				deploymentPrefix = "upg-old"
				variables = make(map[string]interface{})
				variables["defuser_name"] = "pgadmin"
				variables["defuser_password"] = "admin"
				opDefs = nil
			})
			It("Successfully upgrades from old", AssertUpgradeSuccessful())
		})
		Context("Upgrading from minor-no-copy version", func() {
			BeforeEach(func() {
				version = versions.GetOldVersion()
				deploymentPrefix = "upg-old-nocopy"
				variables = make(map[string]interface{})
				variables["defuser_name"] = "pgadmin"
				variables["defuser_password"] = "admin"
				opDefs = helpers.Define_upgrade_no_copy_ops()
			})
			It("Successfully upgrades from old with no copy of the data directory", AssertUpgradeSuccessful())
		})
		Context("Upgrading from master version", func() {
			BeforeEach(func() {
				version = versions.GetLatestVersion()
				deploymentPrefix = "upg-master"
				variables = make(map[string]interface{})
				variables["defuser_name"] = "pgadmin"
				variables["defuser_password"] = "admin"
				opDefs = nil
			})
			It("Successfully upgrades from master", AssertUpgradeSuccessful())
		})
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
		err = director.GetEnv(envName).DeleteDeployment()
		Expect(err).NotTo(HaveOccurred())
	})

})
