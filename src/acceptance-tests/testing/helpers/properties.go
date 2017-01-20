package helpers

import (
	"errors"
	"fmt"

	yaml "gopkg.in/yaml.v2"
)

type Properties struct {
	Databases PgProperties `yaml:"databases"`
}
type PgProperties struct {
	Address               string                `yaml:"address,omitempty"`
	Databases             []PgDBProperties      `yaml:"databases,omitempty"`
	Roles                 []PgRoleProperties    `yaml:"roles,omitempty"`
	Port                  int                   `yaml:"port"`
	MaxConnections        int                   `yaml:"max_connections"`
	LogLinePrefix         string                `yaml:"log_line_prefix"`
	CollectStatementStats bool                  `yaml:"collect_statement_statistics"`
	MonitTimeout          int                   `yaml:"monit_timeout,omitempty"`
	AdditionalConfig      PgAdditionalConfigMap `yaml:"additional_config,omitempty"`
}

type PgDBProperties struct {
	CITExt bool   `yaml:"citext"`
	Name   string `yaml:"name"`
	Tag    string `yaml:"tag"`
}

type PgRoleProperties struct {
	Name        string   `yaml:"name"`
	Password    string   `yaml:"password"`
	Tag         string   `yaml:"tag"`
	Permissions []string `yaml:"permissions,omitempty"`
}

type PgAdditionalConfig interface{}
type PgAdditionalConfigMap map[string]PgAdditionalConfig

var defaultPgProperties = PgProperties{
	LogLinePrefix:         "%m: ",
	CollectStatementStats: false,
	MaxConnections:        500,
}

const MissingMandatoryProp = "Mandatory property is missing"

func LoadProperties(yamlData []byte) (Properties, error) {
	var props Properties
	var err error

	props = Properties{Databases: defaultPgProperties}
	err = yaml.Unmarshal(yamlData, &props)
	if err != nil {
		return Properties{}, err
	}
	if props.Databases.Port == 0 {
		return Properties{}, errors.New(MissingMandatoryProp)
	}
	return props, nil
}

func (pp Properties) GetPostgresURL(address string) string {
	var result string
	pgp := pp.Databases
	if address == "" {
		address = pgp.Address
	}

	//DATABASE_URL="postgres://${PG_USER}:${PG_PSW}@${PG_HOST}:${PG_PORT}/${PG_DB}"
	result = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", pgp.Roles[0].Name, pgp.Roles[0].Password, address, pgp.Port, "postgres")
	return result
}
