package helpers

import (
	"io/ioutil"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

var defaultVersionsFile string = "../versions.yml"

type PostgresReleaseVersions struct {
	sortedKeys []int
	Versions   map[int]string `yaml:"versions"`
	Old        int            `yaml:"old"`
}

func NewPostgresReleaseVersions(versionFile string) (PostgresReleaseVersions, error) {
	var versions PostgresReleaseVersions

	if versionFile == "" {
		versionFile = defaultVersionsFile
	}

	data, err := ioutil.ReadFile(versionFile)
	if err != nil {
		return PostgresReleaseVersions{}, err
	}
	if err := yaml.Unmarshal(data, &versions); err != nil {
		return PostgresReleaseVersions{}, err
	}
	for key := range versions.Versions {
		versions.sortedKeys = append(versions.sortedKeys, key)
	}
	sort.Ints(versions.sortedKeys)

	return versions, nil
}

func (v PostgresReleaseVersions) GetOldVersion() int {
	return v.Old
}

func (v PostgresReleaseVersions) GetLatestVersion() int {
	return v.sortedKeys[len(v.sortedKeys)-1]
}

func (v PostgresReleaseVersions) GetPostgreSQLVersion(key int) string {
	return v.Versions[key]
}

func (v PostgresReleaseVersions) IsMajor(current string, key int) bool {
	value1 := strings.Split(v.Versions[key], ".")
	value2 := strings.Split(current, ".")
	if strings.Join(value1[:len(value1)-1], ".") == strings.Join(value2[:len(value2)-1], ".") {
		return false
	}
	return true
}
