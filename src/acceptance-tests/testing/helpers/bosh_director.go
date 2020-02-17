package helpers

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
	boshtempl "github.com/cloudfoundry/bosh-cli/director/template"
	boshuaa "github.com/cloudfoundry/bosh-cli/uaa"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	cfgtypes "github.com/cloudfoundry/config-server/types"
	patch "github.com/cppforlife/go-patch/patch"
)

type BOSHDirector struct {
	Uaa                    boshuaa.UAA
	Director               boshdir.Director
	DeploymentsInfo        map[string]*DeploymentData
	DirectorConfig         BOSHConfig
	CloudConfig            BOSHCloudConfig
	DefaultReleasesVersion map[string]string
}
type DeploymentData struct {
	ManifestBytes []byte
	ManifestData  map[string]interface{}
	Deployment    boshdir.Deployment
	Variables     boshtempl.Variables
}
type BOSHConfig struct {
	Target      string          `yaml:"target"`
	Credentials BOSHCredentials `yaml:"credentials"`
	UseUaa      bool            `yaml:"use_uaa"`
}

type BOSHCredentials struct {
	Client       string `yaml:"client"`
	ClientSecret string `yaml:"client_secret"`
	CACert       string `yaml:"ca_cert"`
}

type BOSHCloudConfig struct {
	AZs                []string         `yaml:"default_azs"`
	Networks           []BOSHJobNetwork `yaml:"default_networks"`
	PersistentDiskType string           `yaml:"default_persistent_disk_type"`
	VmType             string           `yaml:"default_vm_type"`
	StemcellOs         string           `yaml:"default_stemcell_os"`
	StemcellVersion    string           `yaml:"default_stemcell_version"`
}
type BOSHJobNetwork struct {
	Name      string   `yaml:"name"`
	StaticIPs []string `yaml:"static_ips,omitempty"`
	Default   []string `yaml:"default,omitempty"`
}

var DefaultBOSHConfig = BOSHConfig{
	Target: "192.168.50.4",
}
var DefaultCloudConfig = BOSHCloudConfig{
	AZs: []string{"z1"},
	Networks: []BOSHJobNetwork{
		BOSHJobNetwork{
			Name: "default",
		},
	},
	PersistentDiskType: "10GB",
	VmType:             "small",
	StemcellOs:         "ubuntu-xenial",
	StemcellVersion:    "latest",
}

type VarsCertLoader struct {
	vars boshtempl.Variables
}

type EvaluateOptions boshtempl.EvaluateOpts
type OpDefinition patch.OpDefinition

const MissingDeploymentNameMsg = "Invalid manifest: deployment name not present"
const VMNotPresentMsg = "No VM exists with name %s"
const ProcessNotPresentInVmMsg = "Process %s does not exist in vm %s"

func GenerateEnvName(prefix string) string {
	return fmt.Sprintf("pgats-%s-%s", prefix, GetUUID())
}

func NewBOSHDirector(boshConfig BOSHConfig, cloudConfig BOSHCloudConfig, releasesVersions map[string]string) (BOSHDirector, error) {
	var boshDirector BOSHDirector
	var uaaURL, directorURL string

	boshDirector.DirectorConfig = boshConfig
	boshDirector.CloudConfig = cloudConfig
	boshDirector.DefaultReleasesVersion = releasesVersions

	directorURL = fmt.Sprintf("https://%s:25555", boshConfig.Target)
	logger := boshlog.NewLogger(boshlog.LevelError)
	factory := boshdir.NewFactory(logger)
	directorConfig, err := boshdir.NewConfigFromURL(directorURL)
	if err != nil {
		return BOSHDirector{}, err
	}
	directorConfig.CACert = boshConfig.Credentials.CACert

	if boshConfig.UseUaa {
		uaaURL = fmt.Sprintf("https://%s:8443", boshConfig.Target)
		uaaFactory := boshuaa.NewFactory(logger)
		uaaConfig, err := boshuaa.NewConfigFromURL(uaaURL)
		if err != nil {
			return BOSHDirector{}, err
		}
		uaaConfig.Client = boshConfig.Credentials.Client
		uaaConfig.ClientSecret = boshConfig.Credentials.ClientSecret
		uaaConfig.CACert = boshConfig.Credentials.CACert
		uaa, err := uaaFactory.New(uaaConfig)
		if err != nil {
			return BOSHDirector{}, err
		}
		boshDirector.Uaa = uaa

		directorConfig.TokenFunc = boshuaa.NewClientTokenSession(uaa).TokenFunc
	} else {
		directorConfig.Client = boshConfig.Credentials.Client
		directorConfig.ClientSecret = boshConfig.Credentials.ClientSecret
	}
	director, err := factory.New(directorConfig, boshdir.NewNoopTaskReporter(), boshdir.NewNoopFileReporter())
	if err != nil {
		return BOSHDirector{}, err
	}

	boshDirector.Director = director
	boshDirector.DeploymentsInfo = make(map[string]*DeploymentData)

	return boshDirector, nil
}

func (bd BOSHDirector) GetEnv(envName string) *DeploymentData {
	return bd.DeploymentsInfo[envName]
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

	if dd.ManifestData["stemcells"] != nil {
		for _, elem := range dd.ManifestData["stemcells"].([]interface{}) {
			stemcellAlias := elem.(map[interface{}]interface{})["alias"]
			if stemcellAlias.(string) == "linux" {
				elem.(map[interface{}]interface{})["os"] = bd.CloudConfig.StemcellOs
				elem.(map[interface{}]interface{})["version"] = bd.CloudConfig.StemcellVersion
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
	bd.DeploymentsInfo[deploymentName] = &dd
	return nil
}
func (bd BOSHDirector) UploadPostgresReleaseFromURL(version int) error {
	return bd.UploadReleaseFromURL("cloudfoundry", "postgres-release", version)
}
func (bd BOSHDirector) UploadReleaseFromURL(organization string, repo string, version int) error {
	url := fmt.Sprintf("https://bosh.io/d/github.com/%s/%s?v=%d", organization, repo, version)
	return bd.Director.UploadReleaseURL(url, "", false, false)
}
func (bd BOSHDirector) UploadLatestReleaseFromURL(organization string, repo string) error {
	url := fmt.Sprintf("https://bosh.io/d/github.com/%s/%s", organization, repo)
	return bd.Director.UploadReleaseURL(url, "", false, false)
}

func (dd DeploymentData) ContainsVariables() bool {
	return dd.ManifestData != nil && dd.ManifestData["variables"] != nil && len(dd.ManifestData["variables"].([]interface{})) != 0
}

func (dd DeploymentData) GetVariable(key string) interface{} {
	if dd.Variables != nil {
		vardef := boshtempl.VariableDefinition{Name: key}
		if value, ok, err := dd.Variables.Get(vardef); ok && err == nil {
			return value
		}
	}
	return nil
}

func (dd *DeploymentData) EvaluateTemplate(vars map[string]interface{}, opDefs []OpDefinition, opts EvaluateOptions) error {
	template := boshtempl.NewTemplate(dd.ManifestBytes)

	var ops patch.Ops
	var opDefinitions []patch.OpDefinition
	for _, def := range opDefs {
		opDefinitions = append(opDefinitions, patch.OpDefinition(def))
	}
	ops, err := patch.NewOpsFromDefinitions(opDefinitions)
	if err != nil {
		return err
	}

	var staticVariables boshtempl.StaticVariables
	var structVariables boshtempl.StaticVariables
	var mapVariables MapVariables

	staticVariables = boshtempl.StaticVariables(vars)
	structVariables = boshtempl.StaticVariables(make(map[string]interface{}))
	result, err := template.Evaluate(boshtempl.StaticVariables(vars), ops, boshtempl.EvaluateOpts(opts))
	if err != nil {
		return err
	}
	dd.ManifestBytes = result
	if err := yaml.Unmarshal(dd.ManifestBytes, &dd.ManifestData); err != nil {
		return err
	}
	multiVars := boshtempl.NewMultiVars([]boshtempl.Variables{staticVariables, structVariables})
	factory := cfgtypes.NewValueGeneratorConcrete(NewVarsCertLoader(multiVars))

	if dd.ManifestData["variables"] != nil {
		for _, elem := range dd.ManifestData["variables"].([]interface{}) {
			vdname := elem.(map[interface{}]interface{})["name"]
			vdtype := elem.(map[interface{}]interface{})["type"]
			vdoptions := elem.(map[interface{}]interface{})["options"]

			generator, err := factory.GetGenerator(vdtype.(string))
			if err != nil {
				return err
			}
			value, err := generator.Generate(vdoptions)
			if err != nil {
				return err
			}
			if vdtype == "ssh" || vdtype == "certificate" {
				structVariables[vdname.(string)] = value
			} else {
				staticVariables[vdname.(string)] = value
			}
		}
	}
	for key, value := range structVariables {
		mapVariables.Add(key, value)
	}
	multiVars = boshtempl.NewMultiVars([]boshtempl.Variables{staticVariables, mapVariables})
	result, err = template.Evaluate(multiVars, ops, boshtempl.EvaluateOpts(opts))
	if err != nil {
		return err
	}
	dd.ManifestBytes = result
	if err := yaml.Unmarshal(dd.ManifestBytes, &dd.ManifestData); err != nil {
		return err
	}
	dd.ManifestData["variables"] = []boshtempl.Variables{}
	dd.ManifestBytes, err = yaml.Marshal(&dd.ManifestData)
	if err != nil {
		return err
	}
	dd.Variables = multiVars
	return nil
}
func (dd DeploymentData) CreateOrUpdateDeployment() error {
	updateOpts := boshdir.UpdateOpts{}
	return dd.Deployment.Update(dd.ManifestBytes, updateOpts)
}

func (dd DeploymentData) PrintDeploymentDiffs() error {
	diff, err := dd.Deployment.Diff(dd.ManifestBytes, false)
	fmt.Println("Deployment differences:")
	for _, line := range diff.Diff {
		lineMod, _ := line[1].(string)

		if lineMod == "added" {
			fmt.Printf("+ %s\n", line[0])
		} else if lineMod == "removed" {
			fmt.Printf("- %s\n", line[0])
		} else {
			fmt.Printf("  %s\n", line[0])
		}
	}
	return err
}

func (dd DeploymentData) DeleteDeployment() error {
	return dd.Deployment.Delete(true)
}

func (dd DeploymentData) Restart(instanceGroupName string) error {
	slug := boshdir.NewAllOrInstanceGroupOrInstanceSlug(instanceGroupName, "0")
	restartOptions := boshdir.RestartOpts{}
	return dd.Deployment.Restart(slug, restartOptions)
}
func (dd DeploymentData) Stop(instanceGroupName string) error {
	slug := boshdir.NewAllOrInstanceGroupOrInstanceSlug(instanceGroupName, "0")
	stopOptions := boshdir.StopOpts{}
	return dd.Deployment.Stop(slug, stopOptions)
}
func (dd DeploymentData) Start(instanceGroupName string) error {
	slug := boshdir.NewAllOrInstanceGroupOrInstanceSlug(instanceGroupName, "0")
	startOptions := boshdir.StartOpts{}
	return dd.Deployment.Start(slug, startOptions)
}

//func (dd DeploymentData) IsVmRunning(vmid string) (bool, error) {
//	return dd.IsVmProcessRunning(vmid, "")
//}
func (dd DeploymentData) IsVmProcessRunning(vmid string, processName string) (bool, error) {
	vms, err := dd.Deployment.VMInfos()
	if err != nil {
		return false, err
	}
	for _, info := range vms {
		if info.ID == vmid {
			if processName == "" {
				return info.IsRunning(), nil
			} else if info.Processes == nil || len(info.Processes) == 0 {
				return false, nil
			} else {
				for _, p := range info.Processes {
					if p.Name == processName {
						return p.IsRunning(), nil
					}
				}
				return false, errors.New(fmt.Sprintf(ProcessNotPresentInVmMsg, processName, vmid))
			}
		}
	}
	return false, errors.New(fmt.Sprintf(VMNotPresentMsg, vmid))
}
func (dd DeploymentData) GetVmAddresses(vmname string) ([]string, error) {
	var result []string
	vms, err := dd.Deployment.VMInfos()
	if err != nil {
		return nil, err
	}
	for _, info := range vms {
		if info.JobName == vmname {
			result = append(result, info.IPs[0])
		}
	}
	if result == nil {
		return nil, errors.New(fmt.Sprintf(VMNotPresentMsg, vmname))
	}
	return result, nil
}
func (dd DeploymentData) GetVmDNS(vmname string) (string, error) {
	var result string
	vms, err := dd.Deployment.VMInfos()
	if err != nil {
		return "", err
	}
	for _, info := range vms {
		if info.JobName == vmname && len(info.DNS) > 0 {
			return info.DNS[0], nil
		}
	}
	if result == "" {
		return "", errors.New(fmt.Sprintf(VMNotPresentMsg, vmname))
	}
	return result, nil
}
func (dd DeploymentData) GetVmAddress(vmname string) (string, error) {
	addresses, err := dd.GetVmAddresses(vmname)
	if err != nil {
		return "", err
	}
	return addresses[0], nil
}
func (dd DeploymentData) GetVmIdByAddress(vmaddress string) (string, error) {
	vms, err := dd.Deployment.VMInfos()
	if err != nil {
		return "", err
	}
	for _, info := range vms {
		for _, ip := range info.IPs {
			if ip == vmaddress {
				return info.ID, nil
			}
		}
	}
	return "", errors.New(fmt.Sprintf(VMNotPresentMsg, vmaddress))
}
func (dd DeploymentData) UpdateResurrection(enable bool) error {
	vms, err := dd.Deployment.VMInfos()
	if err != nil {
		return err
	}
	for _, info := range vms {
		err = dd.Deployment.EnableResurrection(boshdir.NewInstanceSlug(info.JobName, info.ID), enable)
		if err != nil {
			return err
		}
	}
	return nil
}
func (dd DeploymentData) GetJobsProperties() (ManifestProperties, error) {
	// since global properties and instance group properties are deprecated, we only considers those specified for the instance group jobs
	var result ManifestProperties
	if dd.ManifestData["instance_groups"] != nil {
		for _, elem := range dd.ManifestData["instance_groups"].([]interface{}) {
			if elem.(map[interface{}]interface{})["jobs"] != nil {
				for _, job := range elem.(map[interface{}]interface{})["jobs"].([]interface{}) {
					bytes, err := yaml.Marshal(job.(map[interface{}]interface{})["properties"])
					if err != nil {
						return ManifestProperties{}, err
					}
					jobInstanceName := job.(map[interface{}]interface{})["name"]
					err = result.LoadJobProperties(jobInstanceName.(string), bytes)
					if err != nil {
						return ManifestProperties{}, err
					}
				}
			}
		}
	}
	return result, nil
}

func NewVarsCertLoader(vars boshtempl.Variables) VarsCertLoader {
	return VarsCertLoader{vars}
}

func (l VarsCertLoader) LoadCerts(name string) (*x509.Certificate, *rsa.PrivateKey, error) {
	val, found, err := l.vars.Get(boshtempl.VariableDefinition{Name: name})
	if err != nil {
		return nil, nil, err
	} else if !found {
		return nil, nil, fmt.Errorf("Expected to find variable '%s' with a certificate", name)
	}

	// Convert to YAML for easier struct parsing
	valBytes, err := yaml.Marshal(val)
	if err != nil {
		return nil, nil, bosherr.WrapErrorf(err, "Expected variable '%s' to be serializable", name)
	}

	type CertVal struct {
		Certificate string
		PrivateKey  string `yaml:"private_key"`
	}

	var certVal CertVal

	err = yaml.Unmarshal(valBytes, &certVal)
	if err != nil {
		return nil, nil, bosherr.WrapErrorf(err, "Expected variable '%s' to be deserializable", name)
	}

	crt, err := l.parseCertificate(certVal.Certificate)
	if err != nil {
		return nil, nil, err
	}

	key, err := l.parsePrivateKey(certVal.PrivateKey)
	if err != nil {
		return nil, nil, err
	}

	return crt, key, nil
}

func (VarsCertLoader) parseCertificate(data string) (*x509.Certificate, error) {
	cpb, _ := pem.Decode([]byte(data))
	if cpb == nil {
		return nil, bosherr.Error("Certificate did not contain PEM formatted block")
	}

	crt, err := x509.ParseCertificate(cpb.Bytes)
	if err != nil {
		return nil, bosherr.WrapError(err, "Parsing certificate")
	}

	return crt, nil
}

func (VarsCertLoader) parsePrivateKey(data string) (*rsa.PrivateKey, error) {
	kpb, _ := pem.Decode([]byte(data))
	if kpb == nil {
		return nil, bosherr.Error("Private key did not contain PEM formatted block")
	}

	key, err := x509.ParsePKCS1PrivateKey(kpb.Bytes)
	if err != nil {
		return nil, bosherr.WrapError(err, "Parsing private key")
	}

	return key, nil
}
