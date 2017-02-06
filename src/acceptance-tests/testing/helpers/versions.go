package helpers

import (
	"io/ioutil"
	"sort"
	"strconv"

	yaml "gopkg.in/yaml.v2"
)

var defaultVersionsFile string = "../versions.yml"

type PostgresReleaseVersions struct {
	sortedKeys []string
	Versions   map[string]string `yaml:"versions"`
	Old        string            `yaml:"old"`
	Older      string            `yaml:"older"`
}
type VersionsSorter []string

func (a VersionsSorter) Len() int      { return len(a) }
func (a VersionsSorter) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a VersionsSorter) Less(i, j int) bool {
	ai, _ := strconv.Atoi(a[i])
	aj, _ := strconv.Atoi(a[j])
	return ai < aj
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
	sort.Sort(VersionsSorter(versions.sortedKeys))

	return versions, nil
}

func (v PostgresReleaseVersions) GetOldVersion() string {
	return v.Old
}

func (v PostgresReleaseVersions) GetOlderVersion() string {
	return v.Older
}
func (v PostgresReleaseVersions) GetLatestVersion() string {
	return v.sortedKeys[len(v.sortedKeys)-1]
}
func (v PostgresReleaseVersions) GetPostgreSQLVersion(key string) string {
	return v.Versions[key]
}
