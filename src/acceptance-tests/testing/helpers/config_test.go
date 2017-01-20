package helpers_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func writeConfigFile(data string) (string, error) {
	tempFile, err := ioutil.TempFile("", "config")
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(tempFile.Name(), []byte(data), os.ModePerm)
	if err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

var _ = Describe("Configuration", func() {
	Describe("Load configuration", func() {
		Context("With a valid config file", func() {
			var configFilePath string
			var cloudConfigPath string

			BeforeEach(func() {
				var err error
				cloudConfigPath, err = writeConfigFile("info: some-info")
				Expect(err).NotTo(HaveOccurred())

				var data = `
target: some-target
username: some-username
password: some-password
director_ca_cert: some-ca-cert
cloud_config_path: %s
`
				configFilePath, err = writeConfigFile(fmt.Sprintf(data, cloudConfigPath))
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				err := os.Remove(configFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("Load the yaml content from the provided path", func() {
				config, err := helpers.LoadConfig(configFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(config).To(Equal(helpers.Config{
					Target:          "some-target",
					Username:        "some-username",
					Password:        "some-password",
					DirectorCACert:  "some-ca-cert",
					CloudConfigPath: cloudConfigPath,
					CloudConfig:     []byte("info: some-info"),
				}))
			})
			It("Get the proper CloudConfig file content", func() {
				config, err := helpers.LoadConfig(configFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(config.CloudConfig).To(Equal([]byte("info: some-info")))
			})
		})

		Context("With an invalid config yaml location", func() {
			It("Should return an error that the file does not exist", func() {
				_, err := helpers.LoadConfig("notExistentPath")
				Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
			})
		})

		Context("With an incorrect config yaml content", func() {
			var configFilePath string

			AfterEach(func() {
				err := os.Remove(configFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("Should return an error if not a valid yaml", func() {
				var err error
				configFilePath, err = writeConfigFile("%%%")
				Expect(err).NotTo(HaveOccurred())

				_, err = helpers.LoadConfig(configFilePath)
				Expect(err).To(MatchError(ContainSubstring("yaml: could not find expected directive name")))
			})

			It("Should return an error if BOSH target missing", func() {
				var err error
				configFilePath, err = writeConfigFile(`{
						"username": "some-username",
						"password": "some-password"
					}`)
				Expect(err).NotTo(HaveOccurred())

				_, err = helpers.LoadConfig(configFilePath)
				Expect(err).To(MatchError(errors.New(helpers.MissingTargetMsg)))
			})

			It("Should return an error if BOSH username missing", func() {
				var err error
				configFilePath, err = writeConfigFile(`{
						"target": "some-target",
						"password": "some-password"
					}`)
				Expect(err).NotTo(HaveOccurred())

				_, err = helpers.LoadConfig(configFilePath)
				Expect(err).To(MatchError(errors.New(helpers.MissingUsernameMsg)))
			})

			It("Should return an error if BOSH password missing", func() {
				var err error
				configFilePath, err = writeConfigFile(`{
						"target": "some-target",
						"username": "some-username"
					}`)
				Expect(err).NotTo(HaveOccurred())

				_, err = helpers.LoadConfig(configFilePath)
				Expect(err).To(MatchError(errors.New(helpers.MissingPasswordMsg)))
			})

			It("Should return an error if given cloud config does not exist", func() {
				var err error
				configFilePath, err = writeConfigFile(`{
						"target": "some-target",
						"username": "some-username",
						"password": "some-password",
						"cloud_config_path": "/notexistentpath"
					}`)
				Expect(err).NotTo(HaveOccurred())

				_, err = helpers.LoadConfig(configFilePath)
				Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
			})

			It("Should return an error if given cloud config is not a valid yml", func() {
				var err error
				cloudConfigPath, err := writeConfigFile("%%%")
				Expect(err).NotTo(HaveOccurred())

				data := `{
						"target": "some-target",
						"username": "some-username",
						"password": "some-password",
						"cloud_config_path": "%s"
					}`

				configFilePath, err = writeConfigFile(fmt.Sprintf(data, cloudConfigPath))
				Expect(err).NotTo(HaveOccurred())

				_, err = helpers.LoadConfig(configFilePath)
				Expect(err).To(MatchError(ContainSubstring("yaml: could not find expected directive name")))
			})
		})
	})

	Describe("PostgresReleaseVersion", func() {

		It("retrieves the postgres_release version from the env if set", func() {
			os.Setenv("POSTGRES_RELEASE_VERSION", "v999.9")
			version := helpers.PostgresReleaseVersion()
			Expect(version).To(Equal("v999.9"))
		})

		It("use default for the postgres_release version if not set", func() {
			os.Setenv("POSTGRES_RELEASE_VERSION", "")
			version := helpers.PostgresReleaseVersion()
			Expect(version).To(Equal("latest"))
		})
	})

	Describe("ConfigPath", func() {
		var configPath string

		BeforeEach(func() {
			configPath = os.Getenv("PGATS_CONFIG")
		})

		AfterEach(func() {
			os.Setenv("PGATS_CONFIG", configPath)
		})

		Context("when a valid path is set", func() {
			It("returns the path", func() {
				os.Setenv("PGATS_CONFIG", "/tmp/some-config.json")
				path, err := helpers.ConfigPath()
				Expect(err).NotTo(HaveOccurred())
				Expect(path).To(Equal("/tmp/some-config.json"))
			})
		})

		Context("when path is not set", func() {
			It("returns an error", func() {
				os.Setenv("PGATS_CONFIG", "")
				_, err := helpers.ConfigPath()
				Expect(err).To(MatchError(fmt.Errorf(helpers.IncorrectEnvMsg, "")))
			})
		})

		Context("when the path is not absolute", func() {
			It("returns an error", func() {
				os.Setenv("PGATS_CONFIG", "some/path.json")
				_, err := helpers.ConfigPath()
				Expect(err).To(MatchError(fmt.Errorf(helpers.IncorrectEnvMsg, "some/path.json")))
			})
		})
	})
})
