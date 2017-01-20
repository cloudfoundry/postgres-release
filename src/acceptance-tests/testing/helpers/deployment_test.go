package helpers_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func writeManifestFile(data string) (string, error) {
	tempFile, err := ioutil.TempFile("", "manifest")
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(tempFile.Name(), []byte(data), os.ModePerm)
	if err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

var _ = Describe("Deployment", func() {
	var (
		director       *fakedir.FakeDirector
		deploymentData helpers.DeploymentData
	)
	Describe("Initialize deployment from manifest", func() {
		BeforeEach(func() {
			director = &fakedir.FakeDirector{}
		})

		Context("With non existent manifest", func() {

			It("Should return an error if not existent manifest", func() {
				var err error
				deploymentData, err = helpers.InitializeDeploymentFromManifestFile("/Not/existent/path", director)
				Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
			})
		})
		Context("With invalid manifest", func() {
			var (
				manifestFilePath string
			)

			AfterEach(func() {
				err := os.Remove(manifestFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("Should return an error if manifest is not a valid yaml", func() {
				var err error
				manifestFilePath, err = writeConfigFile("%%%")
				Expect(err).NotTo(HaveOccurred())
				deploymentData, err = helpers.InitializeDeploymentFromManifestFile(manifestFilePath, director)
				Expect(err).To(MatchError(ContainSubstring("yaml: could not find expected directive name")))
			})

			It("Should return an error if manifest has no deployment name", func() {
				var err error
				data := `
director_uuid: <%= %x[bosh status --uuid] %>
stemcells:
- alias: linux
  name: bosh-warden-boshlite-ubuntu-trusty-go_agent
  version: latest
`
				manifestFilePath, err = writeConfigFile(data)
				Expect(err).NotTo(HaveOccurred())
				deploymentData, err = helpers.InitializeDeploymentFromManifestFile(manifestFilePath, director)
				Expect(err).To(MatchError(errors.New(helpers.MissingDeploymentNameMsg)))
			})
		})

		Context("With a valid manifest", func() {
			var manifestFilePath string

			BeforeEach(func() {
				var err error
				deploymentFake := &fakedir.FakeDeployment{}
				vmInfoFake := boshdir.VMInfo{
					JobName: "postgres",
					IPs:     []string{"1.1.1.1"},
				}
				deploymentFake.VMInfosReturns([]boshdir.VMInfo{vmInfoFake}, nil)
				director.FindDeploymentReturns(deploymentFake, nil)
				data := `
director_uuid: <%= %x[bosh status --uuid] %>
name: test
jobs:
- name: postgres
  instances: 1
properties:
  databases:
    databases:
    - name: pgdb
    port: 1111
    roles:
    - name: pguser
      password: pgpsw
`
				manifestFilePath, err = writeConfigFile(data)
				Expect(err).NotTo(HaveOccurred())
				deploymentData, err = helpers.InitializeDeploymentFromManifestFile(manifestFilePath, director)
				Expect(err).NotTo(HaveOccurred())
			})
			AfterEach(func() {
				var err error
				err = os.Remove(manifestFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("Successfully create and delete a deployment", func() {
				var err error
				err = deploymentData.CreateOrUpdateDeployment()
				Expect(err).NotTo(HaveOccurred())
				err = deploymentData.DeleteDeployment()
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Access deployed environment", func() {

		var manifestFilePath string

		AfterEach(func() {
			err := os.Remove(manifestFilePath)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("Getting VM information", func() {

			BeforeEach(func() {
				director = &fakedir.FakeDirector{}
				var err error
				data := `
director_uuid: <%= %x[bosh status --uuid] %>
name: test
`
				manifestFilePath, err = writeConfigFile(data)
				Expect(err).NotTo(HaveOccurred())
			})

			It("Should return an error if getting address of a non-existent vm", func() {
				var err error
				deploymentFake := &fakedir.FakeDeployment{}
				deploymentFake.VMInfosReturns([]boshdir.VMInfo{}, nil)
				director.FindDeploymentReturns(deploymentFake, nil)
				deploymentData, err = helpers.InitializeDeploymentFromManifestFile(manifestFilePath, director)
				Expect(err).NotTo(HaveOccurred())
				_, err = deploymentData.GetVmAddress("postgres")
				Expect(err).To(Equal(errors.New(fmt.Sprintf(helpers.VMNotPresentMsg, "postgres"))))

				_, err = deploymentData.GetPostgresURL()
				Expect(err).To(Equal(errors.New(fmt.Sprintf(helpers.VMNotPresentMsg, "postgres"))))
			})
			It("Should return an error if VMInfo fails", func() {
				var err error
				deploymentFake := &fakedir.FakeDeployment{}
				deploymentFake.VMInfosReturns([]boshdir.VMInfo{}, errors.New("fake-error"))
				director.FindDeploymentReturns(deploymentFake, nil)
				deploymentData, err = helpers.InitializeDeploymentFromManifestFile(manifestFilePath, director)
				Expect(err).NotTo(HaveOccurred())
				_, err = deploymentData.GetVmAddress("postgres")
				Expect(err).To(Equal(errors.New("fake-error")))

				_, err = deploymentData.GetPostgresURL()
				Expect(err).To(Equal(errors.New("fake-error")))

			})
			It("Gets the proper vm address", func() {
				var err error
				deploymentFake := &fakedir.FakeDeployment{}
				vmInfoFake := boshdir.VMInfo{
					JobName: "postgres",
					IPs:     []string{"1.1.1.1"},
				}
				deploymentFake.VMInfosReturns([]boshdir.VMInfo{vmInfoFake}, nil)
				director.FindDeploymentReturns(deploymentFake, nil)
				deploymentData, err = helpers.InitializeDeploymentFromManifestFile(manifestFilePath, director)
				Expect(err).NotTo(HaveOccurred())
				address, err := deploymentData.GetVmAddress("postgres")
				Expect(err).NotTo(HaveOccurred())
				Expect(address).To(Equal("1.1.1.1"))
			})

		})
		Context("Getting Postgres information with an invalid manifest", func() {

			BeforeEach(func() {
				director = &fakedir.FakeDirector{}
				var err error
				data := `
director_uuid: <%= %x[bosh status --uuid] %>
name: test
properties:
  databases: ~
`
				manifestFilePath, err = writeConfigFile(data)
				Expect(err).NotTo(HaveOccurred())
				deploymentFake := &fakedir.FakeDeployment{}
				vmInfoFake := boshdir.VMInfo{
					JobName: "postgres",
					IPs:     []string{"1.1.1.1"},
				}
				deploymentFake.VMInfosReturns([]boshdir.VMInfo{vmInfoFake}, nil)
				director.FindDeploymentReturns(deploymentFake, nil)
				deploymentData, err = helpers.InitializeDeploymentFromManifestFile(manifestFilePath, director)
				Expect(err).NotTo(HaveOccurred())
			})

			It("Should return an error if incorrect postgres properties", func() {
				var err error

				_, err = deploymentData.GetPostgresURL()
				Expect(err).To(MatchError(errors.New(helpers.MissingMandatoryProp)))

				_, err = deploymentData.GetPostgresProps()
				Expect(err).To(MatchError(errors.New(helpers.MissingMandatoryProp)))

			})
		})
		Context("Getting Postgres information with a valid manifest", func() {

			BeforeEach(func() {
				director = &fakedir.FakeDirector{}
				var err error
				data := `
director_uuid: <%= %x[bosh status --uuid] %>
name: test
properties:
  databases:
    databases:
    - name: pgdb
    port: 1111
    roles:
    - name: pguser
      password: pgpsw
`
				manifestFilePath, err = writeConfigFile(data)
				Expect(err).NotTo(HaveOccurred())
				deploymentFake := &fakedir.FakeDeployment{}
				vmInfoFake := boshdir.VMInfo{
					JobName: "postgres",
					IPs:     []string{"1.1.1.1"},
				}
				deploymentFake.VMInfosReturns([]boshdir.VMInfo{vmInfoFake}, nil)
				director.FindDeploymentReturns(deploymentFake, nil)
				deploymentData, err = helpers.InitializeDeploymentFromManifestFile(manifestFilePath, director)
				Expect(err).NotTo(HaveOccurred())
			})

			It("Gets the proper postgres url", func() {
				var err error
				url, err := deploymentData.GetPostgresURL()
				Expect(err).NotTo(HaveOccurred())
				Expect(url).To(Equal("postgres://pguser:pgpsw@1.1.1.1:1111/postgres?sslmode=disable"))
			})

			It("Gets the proper postgres props", func() {
				var err error
				props, err := deploymentData.GetPostgresProps()
				Expect(err).NotTo(HaveOccurred())
				expectedProps := helpers.Properties{
					Databases: helpers.PgProperties{
						Databases: []helpers.PgDBProperties{
							{Name: "pgdb"},
						},
						Port:                  1111,
						MaxConnections:        500,
						LogLinePrefix:         "%m: ",
						CollectStatementStats: false,
						Roles: []helpers.PgRoleProperties{
							{Name: "pguser",
								Password: "pgpsw"},
						},
					},
				}
				Expect(props).To(Equal(expectedProps))
			})
		})
	})
})
