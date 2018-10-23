package helpers

import (
	"fmt"
	"strconv"
)

const DeployLatestVersion = -1

type DeployHelper struct {
	director     BOSHDirector
	name         string
	manifestPath string
	pgVersion    int
	variables    map[string]interface{}
	opDefs       []OpDefinition
	printDiffs   bool
	networkName  string
}

func NewDeployHelper(params PgatsConfig, prefix string, pgVersion int) (DeployHelper, error) {
	var deployHelper DeployHelper
	var err error

	releases := make(map[string]string)
	releases["postgres"] = params.PGReleaseVersion

	deployHelper.director, err = NewBOSHDirector(params.Bosh, params.BoshCC, releases)
	if err != nil {
		return DeployHelper{}, err
	}

	deployHelper.manifestPath = "../testing/templates/postgres_simple.yml"
	deployHelper.SetDeploymentName(prefix)
	deployHelper.SetPGVersion(pgVersion)
	deployHelper.networkName = params.BoshCC.Networks[0].Name
	deployHelper.InitializeVariables()
	deployHelper.opDefs = nil
	deployHelper.printDiffs = false
	return deployHelper, nil
}

func (d *DeployHelper) SetDeploymentName(prefix string) {
	d.name = GenerateEnvName(prefix)
}

func (d *DeployHelper) EnablePrintDiffs() {
	d.printDiffs = true
}

func (d *DeployHelper) SetPGVersion(version int) {
	d.pgVersion = version
}

func (d *DeployHelper) InitializeVariables() {
	d.variables = make(map[string]interface{})
	d.variables["defuser_name"] = "pgadmin"
	d.variables["defuser_password"] = "adm$in!"

	d.variables["certs_matching_certs"] = "certuser_matching_certs"
	d.variables["certs_matching_name"] = "certuser_matching_name"

	d.variables["certs_mapped_certs"] = "certuser_mapped_certs"
	d.variables["certs_mapped_name"] = "certuser_mapped_name"
	d.variables["certs_mapped_cn"] = "certuser mapped cn"

	d.variables["certs_wrong_certs"] = "certuser_wrong_certs"
	d.variables["certs_wrong_cn"] = "certuser_wrong_cn"

	d.variables["certs_bad_ca"] = "bad_ca"

	d.variables["superuser_name"] = "superuser"
	d.variables["superuser_password"] = "superpsw"
	d.variables["testuser_name"] = "sshuser"
	d.variables["postgres_dns"] = fmt.Sprintf("q-s0.postgres.%s.%s.bosh", d.networkName, d.name)
}

func (d *DeployHelper) SetVariable(name string, value interface{}) {
	d.variables[name] = value
}

func (d DeployHelper) GetVariable(name string) interface{} {
	return d.variables[name]
}

func (d *DeployHelper) SetOpDefs(opDefs []OpDefinition) {
	d.opDefs = opDefs
}

func (d DeployHelper) GetDeployment() *DeploymentData {
	return d.director.GetEnv(d.name)
}

func (d DeployHelper) GetDeploymentName() string {
	return d.name
}

func (d DeployHelper) UploadLatestReleaseFromURL(organization string, project string) error {
	return d.director.UploadLatestReleaseFromURL(organization, project)
}

func (d DeployHelper) runDeploy() error {
	var err error
	if d.printDiffs {
		err := d.GetDeployment().PrintDeploymentDiffs()
		if err != nil {
			return fmt.Errorf("%v%v", "error printing diffs:", err.Error())
		}
	}
	err = d.GetDeployment().CreateOrUpdateDeployment()
	if err != nil {
		return err
	}
	return nil
}

func (d DeployHelper) Deploy() error {
	var err error
	var vars map[string]interface{}
	if d.variables != nil {
		vars = d.variables
	} else {
		vars = make(map[string]interface{})
	}
	releases := make(map[string]string)
	if d.pgVersion != DeployLatestVersion {
		releases["postgres"] = strconv.Itoa(d.pgVersion)
		err = d.director.UploadPostgresReleaseFromURL(d.pgVersion)
		if err != nil {
			return err
		}
	}

	err = d.director.SetDeploymentFromManifest(d.manifestPath, releases, d.name)
	if err != nil {
		return err
	}

	if d.GetDeployment().ContainsVariables() || d.variables != nil {
		if d.GetDeployment().ContainsVariables() {
			if _, err = d.GetDeployment().GetVmAddress("postgres"); err != nil {

				vars["postgres_host"] = "1.1.1.1"
				err = d.GetDeployment().EvaluateTemplate(vars, d.opDefs, EvaluateOptions{})
				if err != nil {
					return err
				}
				err = d.runDeploy()
				if err != nil {
					return err
				}
			}
			var pgHost string
			pgHost, err = d.GetDeployment().GetVmDNS("postgres")
			if err != nil {
				pgHost, err = d.GetDeployment().GetVmAddress("postgres")
				if err != nil {
					return err
				}
			}
			vars["postgres_host"] = pgHost

			err = d.director.SetDeploymentFromManifest(d.manifestPath, releases, d.name)
			if err != nil {
				return err
			}
		}
		err = d.GetDeployment().EvaluateTemplate(vars, d.opDefs, EvaluateOptions{})
		if err != nil {
			return err
		}
	}
	err = d.runDeploy()
	if err != nil {
		return err
	}

	return nil
}
func (d DeployHelper) GetPostgresJobProps() (Properties, error) {
	var err error
	manifestProps, err := d.GetDeployment().GetJobsProperties()
	if err != nil {
		return Properties{}, err
	}
	pgprops := manifestProps.GetJobProperties("postgres")[0]
	return pgprops, nil
}

func (d DeployHelper) GetPGPropsAndHost() (Properties, string, error) {

	pgprops, err := d.GetPostgresJobProps()
	if err != nil {
		return Properties{}, "", err
	}
	var pgHost string
	pgHost, err = d.GetDeployment().GetVmDNS("postgres")
	if err != nil {
		pgHost, err = d.GetDeployment().GetVmAddress("postgres")
		if err != nil {
			return pgprops, "", err
		}
	}
	return pgprops, pgHost, nil
}

func (d DeployHelper) WriteSSHKey() (string, error) {
	sshKey := d.GetDeployment().GetVariable("sshkey")
	keyPath, err := WriteFile(sshKey.(map[interface{}]interface{})["private_key"].(string))
	if err != nil {
		// set permission to 600
		err = SetPermissions(keyPath, 0600)
	}
	return keyPath, err
}

func (d DeployHelper) ConnectToPostgres(pgHost string, pgprops Properties) (PGData, error) {

	pgc := PGCommon{
		Address: pgHost,
		Port:    pgprops.Databases.Port,
		DefUser: User{
			Name:     d.variables["defuser_name"].(string),
			Password: d.variables["defuser_password"].(string),
		},
		AdminUser: User{
			Name:     d.variables["superuser_name"].(string),
			Password: d.variables["superuser_password"].(string),
		},
		CertUser: User{},
	}
	DB, err := NewPostgres(pgc)
	if err != nil {
		return PGData{}, err
	}
	return DB, nil
}
