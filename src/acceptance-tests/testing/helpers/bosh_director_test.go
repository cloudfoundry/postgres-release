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
		director     helpers.BOSHDirector
		fakeDirector *fakedir.FakeDirector
	)
	BeforeEach(func() {
		fakeDirector = &fakedir.FakeDirector{}
		releases := make(map[string]string)
		releases["postgres"] = "latest"
		director = helpers.BOSHDirector{
			Director:               fakeDirector,
			DirectorConfig:         helpers.DefaultBOSHConfig,
			CloudConfig:            helpers.DefaultCloudConfig,
			DefaultReleasesVersion: releases,
		}
	})

	Describe("Initialize deployment from manifest", func() {
		Context("With non existent manifest", func() {

			It("Should return an error if not existent manifest", func() {
				var err error
				err = director.SetDeploymentFromManifest("/Not/existent/path", nil, "dummy")
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
				err = director.SetDeploymentFromManifest(manifestFilePath, nil, "dummy")
				Expect(err).To(MatchError(ContainSubstring("yaml: could not find expected directive name")))
			})

			It("Should return an error if deployment name not provided in input", func() {
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
				err = director.SetDeploymentFromManifest(manifestFilePath, nil, "")
				Expect(err).To(MatchError(errors.New(helpers.MissingDeploymentNameMsg)))
			})
			It("Properly set the provided deployment name", func() {
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
				err = director.SetDeploymentFromManifest(manifestFilePath, nil, helpers.GenerateEnvName("dummy"))
				Expect(err).NotTo(HaveOccurred())
				name := director.DeploymentInfo.ManifestData["name"]
				Expect(name).To(MatchRegexp("pgats-([\\w-]+)-(.{36})"))
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
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				data := `
director_uuid: xxx
name: test
releases:
- version: "%s"
  name: postgres
instance_groups:
- name: postgres
  instances: 1
  azs: ["%s"]
  networks:
  - name: "%s"
  jobs:
  - name: postgres
    release: postgres
  persistent_disk_type: "%s"
  vm_type: "%s"
  stemcell: linux
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
				err = director.SetDeploymentFromManifest(manifestFilePath, nil, "dummy")
				Expect(err).NotTo(HaveOccurred())
			})
			AfterEach(func() {
				var err error
				err = os.Remove(manifestFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("Successfully create and delete a deployment", func() {
				var err error
				err = director.DeploymentInfo.CreateOrUpdateDeployment()
				Expect(err).NotTo(HaveOccurred())
				err = director.DeploymentInfo.DeleteDeployment()
				Expect(err).NotTo(HaveOccurred())
				// TODO check substition of data from cloud config options
			})
		})
	})

	Describe("Update director", func() {
		Context("Uploading a release", func() {
			It("correctly upload release", func() {
				fakeDirector.UploadReleaseURLReturns(nil)
				err := director.UploadReleaseFromURL("some-org","some-repo","1")
				Expect(err).NotTo(HaveOccurred())
			})
			It("Fail to upload release", func() {
				fakeDirector.UploadReleaseURLReturns(errors.New("fake-error"))
				err := director.UploadReleaseFromURL("some-org","some-repo","1")
				Expect(err).To(Equal(errors.New("fake-error")))
			})
		})
		Context("Uploading postgres release", func() {
			It("Correctly upload release", func() {
				fakeDirector.UploadReleaseURLReturns(nil)
				err := director.UploadPostgresReleaseFromURL("1")
				Expect(err).NotTo(HaveOccurred())
			})
			It("Fail to upload release", func() {
				fakeDirector.UploadReleaseURLReturns(errors.New("fake-error"))
				err := director.UploadPostgresReleaseFromURL("1")
				Expect(err).To(Equal(errors.New("fake-error")))
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
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				err = director.SetDeploymentFromManifest(manifestFilePath, nil, "dummy")
				Expect(err).NotTo(HaveOccurred())
				_, err = director.DeploymentInfo.GetVmAddress("postgres")
				Expect(err).To(Equal(errors.New(fmt.Sprintf(helpers.VMNotPresentMsg, "postgres"))))
			})
			It("Should return an error if VMInfo fails", func() {
				var err error
				deploymentFake := &fakedir.FakeDeployment{}
				deploymentFake.VMInfosReturns([]boshdir.VMInfo{}, errors.New("fake-error"))
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				err = director.SetDeploymentFromManifest(manifestFilePath, nil, "dummy")
				Expect(err).NotTo(HaveOccurred())
				_, err = director.DeploymentInfo.GetVmAddress("postgres")
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
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				err = director.SetDeploymentFromManifest(manifestFilePath, nil, "dummy")
				Expect(err).NotTo(HaveOccurred())
				address, err := director.DeploymentInfo.GetVmAddress("postgres")
				Expect(err).NotTo(HaveOccurred())
				Expect(address).To(Equal("1.1.1.1"))
			})

		})
		Context("Getting Postgres information with an invalid manifest", func() {

			BeforeEach(func() {
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
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				err = director.SetDeploymentFromManifest(manifestFilePath, nil, "dummy")
				Expect(err).NotTo(HaveOccurred())
			})

			It("Should return an error if incorrect postgres properties", func() {
				var err error

				_, err = director.DeploymentInfo.GetPostgresProps()
				Expect(err).To(MatchError(errors.New(helpers.MissingMandatoryProp)))

			})
		})
		Context("Getting Postgres information with a valid manifest", func() {

			BeforeEach(func() {
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
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				err = director.SetDeploymentFromManifest(manifestFilePath, nil, "dummy")
				Expect(err).NotTo(HaveOccurred())
			})

			It("Gets the proper postgres props", func() {
				var err error
				props, err := director.DeploymentInfo.GetPostgresProps()
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
