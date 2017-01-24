package helpers

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

const MissingCertificateMsg = "missing `director_ca_cert` - specify BOSH director CA certificate"
const IncorrectEnvMsg = "$PGATS_CONFIG %q does not specify an absolute path to test config file"

type PgatsConfig struct {
	Bosh             PgatsBoshConfig  `yaml:"bosh"`
	BoshCC           PgatsCloudConfig `yaml:"cloud_configs"`
	PGReleaseVersion string           `yaml:"postgres_release_version"`
}
type PgatsBoshConfig struct {
	Target         string `yaml:"target"`
	Username       string `yaml:"username"`
	Password       string `yaml:"password"`
	DirectorCACert string `yaml:"director_ca_cert"`
}
type PgatsCloudConfig struct {
	AZs                []string          `yaml:"default_azs"`
	Networks           []PgatsJobNetwork `yaml:"default_networks"`
	PersistentDiskType string            `yaml:"default_persistent_disk_type"`
	VmType             string            `yaml:"default_vm_type"`
}
type PgatsJobNetwork struct {
	Name      string   `yaml:"name"`
	StaticIPs []string `yaml:"static_ips,omitempty"`
	Default   []string `yaml:"default,omitempty"`
}

var DefaultPgatsConfig = PgatsConfig{
	Bosh: PgatsBoshConfig{
		Target:   "192.168.50.4",
		Username: "admin",
		Password: "admin",
	},
	BoshCC: PgatsCloudConfig{
		AZs: []string{"z1"},
		Networks: []PgatsJobNetwork{
			PgatsJobNetwork{
				Name: "private",
			},
		},
		PersistentDiskType: "10GB",
		VmType:             "m3.medium",
	},
	PGReleaseVersion: "latest",
}

func LoadConfig(configFilePath string) (PgatsConfig, error) {
	var config PgatsConfig
	config = DefaultPgatsConfig

	configFile, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return PgatsConfig{}, err
	}
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		return PgatsConfig{}, err
	}

	if config.Bosh.DirectorCACert == "" {
		return PgatsConfig{}, errors.New(MissingCertificateMsg)
	}
	return config, nil
}

func ConfigPath() (string, error) {
	path := os.Getenv("PGATS_CONFIG")
	if path == "" || !strings.HasPrefix(path, "/") {
		return "", fmt.Errorf(IncorrectEnvMsg, path)
	}

	return path, nil
}
