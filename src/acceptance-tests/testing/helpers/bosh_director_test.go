package helpers_test

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Deployment", func() {
	var (
		envName      string
		director     helpers.BOSHDirector
		fakeDirector *fakedir.FakeDirector
	)
	BeforeEach(func() {
		envName = helpers.GenerateEnvName("dummy")
		fakeDirector = &fakedir.FakeDirector{}
		releases := make(map[string]string)
		releases["postgres"] = "latest"
		director = helpers.BOSHDirector{
			Director:               fakeDirector,
			DeploymentsInfo:        make(map[string]*helpers.DeploymentData),
			DirectorConfig:         helpers.DefaultBOSHConfig,
			CloudConfig:            helpers.DefaultCloudConfig,
			DefaultReleasesVersion: releases,
		}
	})

	Describe("Initialize deployment from manifest", func() {
		Context("With non existent manifest", func() {

			It("Should return an error if not existent manifest", func() {
				var err error
				err = director.SetDeploymentFromManifest("/Not/existent/path", nil, envName)
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
				manifestFilePath, err = helpers.WriteFile("%%%")
				Expect(err).NotTo(HaveOccurred())
				err = director.SetDeploymentFromManifest(manifestFilePath, nil, envName)
				Expect(err).To(MatchError(ContainSubstring("yaml: could not find expected directive name")))
			})

			It("Should return an error if deployment name not provided in input", func() {
				var err error
				data := `
director_uuid: <%= %x[bosh status --uuid] %>
stemcells:
- alias: linux
  name: bosh-warden-boshlite-ubuntu-xenial-go_agent
  version: latest
`
				manifestFilePath, err = helpers.WriteFile(data)
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
  name: bosh-warden-boshlite-ubuntu-xenial-go_agent
  version: latest
`
				manifestFilePath, err = helpers.WriteFile(data)
				Expect(err).NotTo(HaveOccurred())
				err = director.SetDeploymentFromManifest(manifestFilePath, nil, envName)
				Expect(err).NotTo(HaveOccurred())
				name := director.GetEnv(envName).ManifestData["name"]
				Expect(name).To(MatchRegexp("pgats-([\\w-]+)-(.{36})"))
			})
		})

		Context("With a valid manifest", func() {
			var manifestFilePath string
			var data string

			BeforeEach(func() {
				var err error
				deploymentFake := &fakedir.FakeDeployment{}
				vmInfoFake := boshdir.VMInfo{
					JobName: "postgres",
					IPs:     []string{"1.1.1.1"},
				}
				deploymentFake.VMInfosReturns([]boshdir.VMInfo{vmInfoFake}, nil)
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				data = `director_uuid: xxx
instance_groups:
- azs:
  - %s
  instances: 1
  jobs:
  - name: postgres
    release: postgres
  name: postgres
  networks:
  - name: %s
  persistent_disk_type: %s
  stemcell: linux
  vm_type: %s
name: %s
properties:
  %s: %s
  databases:
    databases:
    - name: pgdb
    port: 1111
    roles:
    - name: pguser
      password: pgpsw
  ssh_key: %s
releases:
- name: postgres
  version: %s
variables: []
`
				input := fmt.Sprintf(data, "xx", "xx", "xx", "xx", "xx", "((key))", "((value))", "((sshkey.public_key))", "xx")
				manifestFilePath, err = helpers.WriteFile(input)
				Expect(err).NotTo(HaveOccurred())
				err = director.SetDeploymentFromManifest(manifestFilePath, nil, envName)
				Expect(err).NotTo(HaveOccurred())
				Expect(director.GetEnv(envName).ContainsVariables()).To(BeFalse())
			})
			AfterEach(func() {
				var err error
				err = os.Remove(manifestFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("Successfully create and delete a deployment", func() {
				var err error
				err = director.GetEnv(envName).CreateOrUpdateDeployment()
				Expect(err).NotTo(HaveOccurred())
				err = director.GetEnv(envName).DeleteDeployment()
				Expect(err).NotTo(HaveOccurred())
			})
			It("Can interpolate variables", func() {
				vars := map[string]interface{}{
					"key":               "aaa",
					"value":             "bar",
					"sshkey.public_key": "key",
				}
				data = fmt.Sprintf(data, "z1", "default", "10GB", "small", envName, vars["key"], vars["value"], vars["sshkey.public_key"], "latest")
				err := director.GetEnv(envName).EvaluateTemplate(vars, nil, helpers.EvaluateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(string(director.GetEnv(envName).ManifestBytes)).To(Equal(data))
			})
			It("Can generate variables", func() {
				var err error
				vars := map[string]interface{}{
					"key":         "foo",
					"common_name": "aaa",
				}
				variables := `variables:
- name: xxx_password
  type: password
- name: sshkey
  type: ssh
- name: xxx_ca
  options:
    common_name: xxx_ca
    is_ca: true
  type: certificate
- name: xxx_cert
  options:
    alternative_names:
    - %s
    ca: xxx_ca
    common_name: %s
  type: certificate
`
				input := fmt.Sprintf(data, "xx", "xx", "xx", "xx", "xx", "((key))", "((xxx_password))", "((sshkey.public_key))", "xx") + fmt.Sprintf(variables, "((common_name))", "((common_name))")
				err = os.Remove(manifestFilePath)
				Expect(err).NotTo(HaveOccurred())
				manifestFilePath, err = helpers.WriteFile(input)
				Expect(err).NotTo(HaveOccurred())
				err = director.SetDeploymentFromManifest(manifestFilePath, nil, envName)
				Expect(err).NotTo(HaveOccurred())
				Expect(director.GetEnv(envName).ContainsVariables()).To(BeTrue())
				err = director.GetEnv(envName).EvaluateTemplate(vars, nil, helpers.EvaluateOptions{})
				Expect(err).NotTo(HaveOccurred())
				props := director.GetEnv(envName).ManifestData["properties"]
				password := props.(map[interface{}]interface{})["foo"]

				sshkey := director.GetEnv(envName).GetVariable("sshkey")
				public_key := sshkey.(map[interface{}]interface{})["public_key"]
				actualDataProperties := director.GetEnv(envName).ManifestData["properties"]
				Expect(strings.TrimSpace(public_key.(string))).To(Equal(strings.TrimSpace(actualDataProperties.(map[interface{}]interface{})["ssh_key"].(string))))
				Expect(strings.TrimSpace(password.(string))).To(Equal(strings.TrimSpace(actualDataProperties.(map[interface{}]interface{})["foo"].(string))))
			})
			It("Fails to interpolate variables", func() {
				vars := map[string]interface{}{
					"key": "foo",
				}
				options := helpers.EvaluateOptions{ExpectAllKeys: true}
				err := director.GetEnv(envName).EvaluateTemplate(vars, nil, options)
				Expect(err).NotTo(BeNil())
			})
			It("Fails to interpolate operation directives", func() {

				var ops []helpers.OpDefinition
				var value interface{}
				value = "aaa"

				helpers.AddOpDefinition(&ops, "replace", "/properties/fake/((key))", value)
				data = fmt.Sprintf(data, "z1", "default", "10GB", "small", envName, "((key))", value, value, "latest")
				err := director.GetEnv(envName).EvaluateTemplate(nil, ops, helpers.EvaluateOptions{})
				Expect(err).NotTo(BeNil())
			})
			It("Can interpolate operation directives", func() {
				var ops []helpers.OpDefinition
				var value interface{}
				value = "aaa"
				helpers.AddOpDefinition(&ops, "replace", "/properties/((key))", value)
				helpers.AddOpDefinition(&ops, "replace", "/properties/ssh_key", value)
				data = fmt.Sprintf(data, "z1", "default", "10GB", "small", envName, "((key))", value, value, "latest")
				err := director.GetEnv(envName).EvaluateTemplate(nil, ops, helpers.EvaluateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(string(director.GetEnv(envName).ManifestBytes)).To(Equal(data))
			})
		})
	})

	Describe("Update director", func() {
		Context("Uploading a release", func() {
			It("correctly upload release", func() {
				fakeDirector.UploadReleaseURLReturns(nil)
				err := director.UploadReleaseFromURL("some-org", "some-repo", 1)
				Expect(err).NotTo(HaveOccurred())
			})
			It("Fail to upload release", func() {
				fakeDirector.UploadReleaseURLReturns(errors.New("fake-error"))
				err := director.UploadReleaseFromURL("some-org", "some-repo", 1)
				Expect(err).To(Equal(errors.New("fake-error")))
			})
		})
		Context("Uploading postgres release", func() {
			It("Correctly upload release", func() {
				fakeDirector.UploadReleaseURLReturns(nil)
				err := director.UploadPostgresReleaseFromURL(1)
				Expect(err).NotTo(HaveOccurred())
			})
			It("Fail to upload release", func() {
				fakeDirector.UploadReleaseURLReturns(errors.New("fake-error"))
				err := director.UploadPostgresReleaseFromURL(1)
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
		Context("Update resurrector for all vms in deployment", func() {
			var deploymentFake *fakedir.FakeDeployment
			BeforeEach(func() {
				var err error
				data := `
director_uuid: <%= %x[bosh status --uuid] %>
name: test
`
				manifestFilePath, err = helpers.WriteFile(data)
				Expect(err).NotTo(HaveOccurred())
				deploymentFake = &fakedir.FakeDeployment{}
				vm1InfoFake := boshdir.VMInfo{
					JobName: "postgres",
					ID:      "xxx-xxx-xxx",
				}
				vm2InfoFake := boshdir.VMInfo{
					JobName: "postgres",
					ID:      "aaa-aaa-aaa-aaa",
				}
				deploymentFake.VMInfosReturns([]boshdir.VMInfo{vm1InfoFake, vm2InfoFake}, nil)
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				err = director.SetDeploymentFromManifest(manifestFilePath, nil, envName)
				Expect(err).NotTo(HaveOccurred())
			})

			It("Fails to restart the VM", func() {
				var err error
				deploymentFake.RestartReturns(errors.New("fake-error"))
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				err = director.GetEnv(envName).Restart("postgres")
				Expect(err).To(Equal(errors.New("fake-error")))
			})

			It("Can restart the VM", func() {
				var err error
				deploymentFake.RestartReturns(nil)
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				err = director.GetEnv(envName).Restart("postgres")
				Expect(err).NotTo(HaveOccurred())
			})

			It("Fails to stop the VM", func() {
				var err error
				deploymentFake.StopReturns(errors.New("fake-error"))
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				err = director.GetEnv(envName).Stop("postgres")
				Expect(err).To(Equal(errors.New("fake-error")))
			})

			It("Can stop the VM", func() {
				var err error
				deploymentFake.StopReturns(nil)
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				err = director.GetEnv(envName).Stop("postgres")
				Expect(err).NotTo(HaveOccurred())
			})
			It("Fails to start the VM", func() {
				var err error
				deploymentFake.StartReturns(errors.New("fake-error"))
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				err = director.GetEnv(envName).Start("postgres")
				Expect(err).To(Equal(errors.New("fake-error")))
			})

			It("Can start the VM", func() {
				var err error
				deploymentFake.StartReturns(nil)
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				err = director.GetEnv(envName).Start("postgres")
				Expect(err).NotTo(HaveOccurred())
			})
			It("Fail to pause resurrection", func() {
				var err error
				deploymentFake.EnableResurrectionReturns(errors.New("fake-error"))
				err = director.GetEnv(envName).UpdateResurrection(false)
				Expect(err).To(Equal(errors.New("fake-error")))
			})
			It("Correctly pause resurrection", func() {
				var err error
				fakeDirector.EnableResurrectionReturns(nil)
				err = director.GetEnv(envName).UpdateResurrection(false)
				Expect(err).NotTo(HaveOccurred())
			})
			It("Fail to print deployment diffs", func() {
				var err error
				expected := boshdir.DeploymentDiff{
					Diff: [][]interface{}{[]interface{}{"name: simple manifest", nil}},
				}
				deploymentFake.DiffReturns(expected, errors.New("fake-error"))
				err = director.GetEnv(envName).PrintDeploymentDiffs()
				Expect(err).To(Equal(errors.New("fake-error")))
			})
			It("Prints the deployment diffs", func() {
				var err error
				expected := boshdir.DeploymentDiff{
					Diff: [][]interface{}{[]interface{}{"name: simple manifest", nil}},
				}
				deploymentFake.DiffReturns(expected, nil)
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				err = director.GetEnv(envName).PrintDeploymentDiffs()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("Getting VM information", func() {

			BeforeEach(func() {
				var err error
				data := `
director_uuid: <%= %x[bosh status --uuid] %>
name: test
`
				manifestFilePath, err = helpers.WriteFile(data)
				Expect(err).NotTo(HaveOccurred())
			})

			It("Correctly checks if a vm is running", func() {
				var err error
				deploymentFake := &fakedir.FakeDeployment{}
				vm1InfoFake := boshdir.VMInfo{
					JobName:      "postgres",
					ID:           "xxx-xxx-xxx",
					ProcessState: "running",
					IPs:          []string{"1.1.1.1"},
					Processes: []boshdir.VMInfoProcess{
						boshdir.VMInfoProcess{
							Name:  "etcd",
							State: "running",
						},
						boshdir.VMInfoProcess{
							Name:  "postgres",
							State: "failed",
						},
					},
				}
				vm2InfoFake := boshdir.VMInfo{
					JobName:      "postgres",
					ID:           "aaa-aaa-aaa-aaa",
					ProcessState: "running",
					IPs:          []string{"2.2.2.2"},
					Processes:    []boshdir.VMInfoProcess{},
				}
				deploymentFake.VMInfosReturns([]boshdir.VMInfo{vm1InfoFake, vm2InfoFake}, nil)
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				err = director.SetDeploymentFromManifest(manifestFilePath, nil, envName)
				Expect(err).NotTo(HaveOccurred())
				//isRunning, err := director.GetEnv(envName).IsVmRunning("aaa-aaa-aaa-aaa")
				//Expect(err).NotTo(HaveOccurred())
				//Expect(isRunning).To(BeTrue())
				//isRunning, err = director.GetEnv(envName).IsVmRunning("xxx-xxx-xxx")
				//Expect(err).NotTo(HaveOccurred())
				//Expect(isRunning).To(BeFalse())
				isRunning, err := director.GetEnv(envName).IsVmProcessRunning("xxx-xxx-xxx", "postgres")
				Expect(err).NotTo(HaveOccurred())
				Expect(isRunning).To(BeFalse())
				isRunning, err = director.GetEnv(envName).IsVmProcessRunning("xxx-xxx-xxx", "etcd")
				Expect(err).NotTo(HaveOccurred())
				Expect(isRunning).To(BeTrue())
				isRunning, err = director.GetEnv(envName).IsVmProcessRunning("aaa-aaa-aaa-aaa", "xxx")
				Expect(err).NotTo(HaveOccurred())
				Expect(isRunning).To(BeFalse())
				_, err = director.GetEnv(envName).IsVmProcessRunning("xxx", "postgres")
				Expect(err).To(Equal(errors.New(fmt.Sprintf(helpers.VMNotPresentMsg, "xxx"))))
				_, err = director.GetEnv(envName).IsVmProcessRunning("xxx-xxx-xxx", "xxx")
				Expect(err).To(Equal(errors.New(fmt.Sprintf(helpers.ProcessNotPresentInVmMsg, "xxx", "xxx-xxx-xxx"))))
			})
			It("Should return an error if getting address of a non-existent vm", func() {
				var err error
				deploymentFake := &fakedir.FakeDeployment{}
				deploymentFake.VMInfosReturns([]boshdir.VMInfo{}, nil)
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				err = director.SetDeploymentFromManifest(manifestFilePath, nil, envName)
				Expect(err).NotTo(HaveOccurred())
				_, err = director.GetEnv(envName).GetVmAddress("postgres")
				Expect(err).To(Equal(errors.New(fmt.Sprintf(helpers.VMNotPresentMsg, "postgres"))))
				_, err = director.GetEnv(envName).GetVmAddresses("postgres")
				Expect(err).To(Equal(errors.New(fmt.Sprintf(helpers.VMNotPresentMsg, "postgres"))))
				_, err = director.GetEnv(envName).GetVmIdByAddress("1.1.1.1")
				Expect(err).To(Equal(errors.New(fmt.Sprintf(helpers.VMNotPresentMsg, "1.1.1.1"))))
			})
			It("Should return an error if VMInfo fails", func() {
				var err error
				deploymentFake := &fakedir.FakeDeployment{}
				deploymentFake.VMInfosReturns([]boshdir.VMInfo{}, errors.New("fake-error"))
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				err = director.SetDeploymentFromManifest(manifestFilePath, nil, envName)
				Expect(err).NotTo(HaveOccurred())
				_, err = director.GetEnv(envName).GetVmAddress("postgres")
				Expect(err).To(Equal(errors.New("fake-error")))
				_, err = director.GetEnv(envName).GetVmAddresses("postgres")
				Expect(err).To(Equal(errors.New("fake-error")))
				_, err = director.GetEnv(envName).GetVmIdByAddress("1.1.1.1")
				Expect(err).To(Equal(errors.New("fake-error")))
			})
			It("Gets the proper vm address and id", func() {
				var err error
				deploymentFake := &fakedir.FakeDeployment{}
				vm1InfoFake := boshdir.VMInfo{
					JobName: "postgres",
					ID:      "xxx-xxx-xxx",
					IPs:     []string{"1.1.1.1"},
				}
				vm2InfoFake := boshdir.VMInfo{
					JobName: "postgres",
					ID:      "aaa-aaa-aaa-aaa",
					IPs:     []string{"2.2.2.2"},
				}
				deploymentFake.VMInfosReturns([]boshdir.VMInfo{vm1InfoFake, vm2InfoFake}, nil)
				fakeDirector.FindDeploymentReturns(deploymentFake, nil)
				err = director.SetDeploymentFromManifest(manifestFilePath, nil, envName)
				Expect(err).NotTo(HaveOccurred())
				address, err := director.GetEnv(envName).GetVmAddress("postgres")
				Expect(err).NotTo(HaveOccurred())
				Expect(address).To(Equal("1.1.1.1"))
				addresses, err := director.GetEnv(envName).GetVmAddresses("postgres")
				Expect(err).NotTo(HaveOccurred())
				Expect(addresses).To(Equal([]string{"1.1.1.1", "2.2.2.2"}))
				uuid, err := director.GetEnv(envName).GetVmIdByAddress(address)
				Expect(err).NotTo(HaveOccurred())
				Expect(uuid).To(Equal(vm1InfoFake.ID))
			})

		})
	})
	Describe("Manage properties", func() {

		var (
			data             string
			manifestFilePath string
		)

		AfterEach(func() {
			err := os.Remove(manifestFilePath)
			Expect(err).NotTo(HaveOccurred())
		})
		JustBeforeEach(func() {
			var err error
			manifestFilePath, err = helpers.WriteFile(data)
			Expect(err).NotTo(HaveOccurred())
			deploymentFake := &fakedir.FakeDeployment{}
			vmInfoFake := boshdir.VMInfo{
				JobName: "postgres",
				IPs:     []string{"1.1.1.1"},
			}
			deploymentFake.VMInfosReturns([]boshdir.VMInfo{vmInfoFake}, nil)
			fakeDirector.FindDeploymentReturns(deploymentFake, nil)
			err = director.SetDeploymentFromManifest(manifestFilePath, nil, envName)
			Expect(err).NotTo(HaveOccurred())
		})
		AssertPropertiesGetSuccessful := func() func() {
			return func() {
				var err error
				props, err := director.GetEnv(envName).GetJobsProperties()
				Expect(err).NotTo(HaveOccurred())
				expectedProps := helpers.Properties{
					Databases: helpers.PgProperties{
						Port: 1111,
						Databases: []helpers.PgDBProperties{
							{Name: "pgdb"},
						},
						MaxConnections:        500,
						LogLinePrefix:         "%m: ",
						CollectStatementStats: false,
						Roles: []helpers.PgRoleProperties{
							{Name: "pguser",
								Password: "pgpsw"},
						},
					},
				}
				Expect(props.GetJobProperties("postgres")).To(Equal([]helpers.Properties{expectedProps}))
			}
		}
		Context("Getting Postgres information from job section", func() {

			BeforeEach(func() {
				data = `
director_uuid: <%= %x[bosh status --uuid] %>
name: test
instance_groups:
- azs:
  - xx
  instances: 1
  jobs:
  - name: postgres
    release: postgres
    properties:
      databases:
        port: 1111
        databases:
        - name: pgdb
        roles:
        - name: pguser
          password: pgpsw
  name: postgres
  networks:
  - name: xx
  persistent_disk_type: xx
  stemcell: linux
  vm_type: xx
`
			})

			It("Correctly gets the proper postgres props", AssertPropertiesGetSuccessful())
		})
	})
})
