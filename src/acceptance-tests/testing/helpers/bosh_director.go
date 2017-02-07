package helpers

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type BOSHDirector struct {
	Director               boshdir.Director
	DeploymentInfo         *DeploymentData
	DirectorConfig         BOSHConfig
	CloudConfig            BOSHCloudConfig
	DefaultReleasesVersion map[string]string
}
type DeploymentData struct {
	ManifestBytes []byte
	ManifestData  map[string]interface{}
	Deployment    boshdir.Deployment
}
type BOSHConfig struct {
	Target         string `yaml:"target"`
	Username       string `yaml:"username"`
	Password       string `yaml:"password"`
	DirectorCACert string `yaml:"director_ca_cert"`
}
type BOSHCloudConfig struct {
	AZs                []string         `yaml:"default_azs"`
	Networks           []BOSHJobNetwork `yaml:"default_networks"`
	PersistentDiskType string           `yaml:"default_persistent_disk_type"`
	VmType             string           `yaml:"default_vm_type"`
}
type BOSHJobNetwork struct {
	Name      string   `yaml:"name"`
	StaticIPs []string `yaml:"static_ips,omitempty"`
	Default   []string `yaml:"default,omitempty"`
}

var DefaultBOSHConfig = BOSHConfig{
	Target:   "192.168.50.4",
	Username: "admin",
	Password: "admin",
}
var DefaultCloudConfig = BOSHCloudConfig{
	AZs: []string{"z1"},
	Networks: []BOSHJobNetwork{
		BOSHJobNetwork{
			Name: "private",
		},
	},
	PersistentDiskType: "10GB",
	VmType:             "m3.medium",
}

const MissingDeploymentNameMsg = "Invalid manifest: deployment name not present"
const VMNotPresentMsg = "No VM exists with name %s"

func GenerateEnvName(prefix string) string {
	guid := "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

	b := make([]byte, 16)
	_, err := rand.Read(b[:])
	if err == nil {
		guid = fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	}
	return fmt.Sprintf("pgats-%s-%s", prefix, guid)
}

func NewBOSHDirector(boshConfig BOSHConfig, cloudConfig BOSHCloudConfig, releasesVersions map[string]string) (BOSHDirector, error) {
	var boshDirector BOSHDirector

	boshDirector.DirectorConfig = boshConfig
	boshDirector.CloudConfig = cloudConfig
	boshDirector.DefaultReleasesVersion = releasesVersions

	directorURL := fmt.Sprintf("https://%s:25555", boshConfig.Target)
	logger := boshlog.NewLogger(boshlog.LevelError)
	factory := boshdir.NewFactory(logger)
	config, err := boshdir.NewConfigFromURL(directorURL)
	if err != nil {
		return BOSHDirector{}, err
	}

	config.Client = boshConfig.Username
	config.ClientSecret = boshConfig.Password
	config.CACert = boshConfig.DirectorCACert

	director, err := factory.New(config, boshdir.NewNoopTaskReporter(), boshdir.NewNoopFileReporter())
	if err != nil {
		return BOSHDirector{}, err
	}
	boshDirector.Director = director
	boshDirector.DeploymentInfo = &DeploymentData{}

	return boshDirector, nil
}

func (bd *BOSHDirector) SetDeploymentFromManifest(manifestFilePath string, releasesVersions map[string]string, deploymentName string) error {
	var err error
	var dd DeploymentData

	dd.ManifestBytes, err = ioutil.ReadFile(manifestFilePath)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(dd.ManifestBytes, &dd.ManifestData); err != nil {
		return err
	}

	dd.ManifestData["name"] = deploymentName

	if dd.ManifestData["releases"] != nil {
		for _, elem := range dd.ManifestData["releases"].([]interface{}) {
			relName := elem.(map[interface{}]interface{})["name"]
			if version, ok := releasesVersions[relName.(string)]; ok {
				elem.(map[interface{}]interface{})["version"] = version
			} else if version, ok := bd.DefaultReleasesVersion[relName.(string)]; ok {
				elem.(map[interface{}]interface{})["version"] = version
			}
		}
	}
	if dd.ManifestData["instance_groups"] != nil {

		netBytes, err := yaml.Marshal(&bd.CloudConfig.Networks)
		if err != nil {
			return err
		}
		var netData []map[string]interface{}
		if err := yaml.Unmarshal(netBytes, &netData); err != nil {
			return err
		}

		for _, elem := range dd.ManifestData["instance_groups"].([]interface{}) {
			elem.(map[interface{}]interface{})["azs"] = bd.CloudConfig.AZs
			elem.(map[interface{}]interface{})["networks"] = netData
			elem.(map[interface{}]interface{})["persistent_disk_type"] = bd.CloudConfig.PersistentDiskType
			elem.(map[interface{}]interface{})["vm_type"] = bd.CloudConfig.VmType
		}
	}

	dd.ManifestBytes, err = yaml.Marshal(&dd.ManifestData)
	if err != nil {
		return err
	}

	if dd.ManifestData["name"] == nil || dd.ManifestData["name"] == "" {
		return errors.New(MissingDeploymentNameMsg)
	}

	dd.Deployment, err = bd.Director.FindDeployment(dd.ManifestData["name"].(string))
	if err != nil {
		return err
	}
	bd.DeploymentInfo = &dd
	return nil
}
func (bd BOSHDirector) UploadPostgresReleaseFromURL(version int) error {
	return bd.UploadReleaseFromURL("cloudfoundry", "postgres-release", version)
}
func (bd BOSHDirector) UploadReleaseFromURL(organization string, repo string, version int) error {
	url := fmt.Sprintf("https://bosh.io/d/github.com/%s/%s?v=%d", organization, repo, version)
	return bd.Director.UploadReleaseURL(url, "", false, false)
}

func (dd DeploymentData) CreateOrUpdateDeployment() error {
	updateOpts := boshdir.UpdateOpts{}
	return dd.Deployment.Update(dd.ManifestBytes, updateOpts)
}

func (dd DeploymentData) DeleteDeployment() error {
	return dd.Deployment.Delete(true)
}

func (dd DeploymentData) GetVmAddress(vmname string) (string, error) {
	var result string
	vms, err := dd.Deployment.VMInfos()
	if err != nil {
		return "", err
	}
	for _, info := range vms {
		if info.JobName == vmname {
			result = info.IPs[0]
		}
	}
	if result == "" {
		return "", errors.New(fmt.Sprintf(VMNotPresentMsg, vmname))
	}
	return result, nil
}
func (dd DeploymentData) GetPostgresProps() (Properties, error) {
	var result Properties
	bytes, err := yaml.Marshal(dd.ManifestData["properties"])
	if err != nil {
		return Properties{}, err
	}
	result, err = LoadProperties(bytes)
	if err != nil {
		return Properties{}, err
	}
	return result, nil
}
