package helpers

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

const MissingTargetMsg = "missing `target` - specify BOSH target"
const MissingPasswordMsg = "missing `password` - specify password for authenticating with BOSH"
const MissingUsernameMsg = "missing `username` - specify username for authenticating with BOSH"
const IncorrectEnvMsg = "$PGATS_CONFIG %q does not specify an absolute path to test config file"

type Config struct {
	Target          string `yaml:"target"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	DirectorCACert  string `yaml:"director_ca_cert,omitempty"`
	CloudConfigPath string `yaml:"cloud_config_path,omitempty"`
	CloudConfig     []byte
}

func LoadConfig(configFilePath string) (Config, error) {
	configFile, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return Config{}, err
	}

	var config Config
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		return Config{}, err
	}

	if config.Target == "" {
		return Config{}, errors.New(MissingTargetMsg)
	}

	if config.Username == "" {
		return Config{}, errors.New(MissingUsernameMsg)
	}

	if config.Password == "" {
		return Config{}, errors.New(MissingPasswordMsg)
	}

	if config.CloudConfigPath != "" {
		config.CloudConfig, err = ioutil.ReadFile(config.CloudConfigPath)
		if err != nil {
			return Config{}, err
		}
		m := make(map[interface{}]interface{})
		err = yaml.Unmarshal(config.CloudConfig, &m)
		if err != nil {
			return Config{}, err
		}
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

func PostgresReleaseVersion() string {
	version := os.Getenv("POSTGRES_RELEASE_VERSION")
	if version == "" {
		version = "latest"
	}
	return version
}
