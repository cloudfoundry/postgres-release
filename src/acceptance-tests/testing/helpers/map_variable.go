package helpers

import (
	"fmt"

	"gopkg.in/yaml.v2"

	boshtempl "github.com/cloudfoundry/bosh-cli/director/template"
)

type MapVariables struct {
	Entries *boshtempl.StaticVariables
}

var _ boshtempl.Variables = MapVariables{}

func (v MapVariables) Get(varDef boshtempl.VariableDefinition) (interface{}, bool, error) {
	if v.Entries == nil {
		return nil, false, nil
	}
	bytes, err := yaml.Marshal(*v.Entries)
	if err != nil {
		return nil, false, err
	}
	result := boshtempl.StaticVariables{}
	err = yaml.Unmarshal(bytes, &result)
	if err != nil {
		fmt.Println("error2", err)
		return nil, false, err
	}
	val, found := result[varDef.Name]
	return val, found, nil
}

func (v *MapVariables) Add(name string, value interface{}) {
	if v.Entries == nil {
		m := boshtempl.StaticVariables(make(map[string]interface{}))
		v.Entries = &m
	}
	(*v.Entries)[name] = value
}
func (v MapVariables) List() ([]boshtempl.VariableDefinition, error) {
	result, err := v.Entries.List()
	return result, err
}
