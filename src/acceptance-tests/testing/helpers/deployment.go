package helpers

import (
	"errors"
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"

	boshman "github.com/cloudfoundry/bosh-cli/deployment/manifest"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type DeploymentData struct {
	ManifestBytes []byte
	ManifestData  boshman.Manifest
	Director      boshdir.Director
	Deployment    boshdir.Deployment
}

const MissingDeploymentNameMsg = "Invalid manifest: deployment name not present"
const VMNotPresentMsg = "No VM exists with name %s"

func TargetDirector(directorURL string, username string, password string, caCert string) (boshdir.Director, error) {

	logger := boshlog.NewLogger(boshlog.LevelError)
	factory := boshdir.NewFactory(logger)
	config, err := boshdir.NewConfigFromURL(directorURL)
	if err != nil {
		return boshdir.DirectorImpl{}, err
	}

	config.Client = username
	config.ClientSecret = password
	config.CACert = caCert

	director, err := factory.New(config, boshdir.NewNoopTaskReporter(), boshdir.NewNoopFileReporter())
	if err != nil {
		return boshdir.DirectorImpl{}, err
	}
	return director, nil
}

func InitializeDeploymentFromManifestFile(manifestFilePath string, director boshdir.Director) (DeploymentData, error) {
	var dd DeploymentData
	var err error
	dd.ManifestBytes, err = ioutil.ReadFile(manifestFilePath)
	if err != nil {
		return DeploymentData{}, err
	}

	if err := yaml.Unmarshal(dd.ManifestBytes, &dd.ManifestData); err != nil {
		return DeploymentData{}, err
	}
	if dd.ManifestData.Name == "" {
		return DeploymentData{}, errors.New(MissingDeploymentNameMsg)
	}

	dd.Deployment, err = director.FindDeployment(dd.ManifestData.Name)
	if err != nil {
		return DeploymentData{}, err
	}

	return dd, nil
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
func (dd DeploymentData) GetPostgresURL() (string, error) {
	var result string
	vmAddress, err := dd.GetVmAddress("postgres")
	if err != nil {
		return "", err
	}
	props, err := dd.GetPostgresProps()
	if err != nil {
		return "", err
	}
	result = props.GetPostgresURL(vmAddress)
	return result, nil
}
func (dd DeploymentData) GetPostgresProps() (Properties, error) {
	var result Properties
	bytes, err := yaml.Marshal(dd.ManifestData.Properties)
	if err != nil {
		return Properties{}, err
	}
	result, err = LoadProperties(bytes)
	if err != nil {
		return Properties{}, err
	}
	return result, nil
}
