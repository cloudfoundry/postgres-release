package helpers

type Janitor struct {
	Timeout  int
	Interval int
	Script   string
}

func AddOpDefinition(ods *[]OpDefinition, defType string, defPath string, defValue interface{}) {
	od := OpDefinition{
		Type:  defType,
		Path:  &defPath,
		Value: &defValue,
	}
	*ods = append(*ods, od)
}

func Define_bbr_ops() []OpDefinition {
	var ops []OpDefinition
	var value interface{}
	var path string

	path = "/instance_groups/name=postgres/jobs/-"
	value = map[interface{}]interface{}{
		"name":    "bbr-postgres-db",
		"release": "postgres",
		"consumes": map[interface{}]interface{}{
			"database": map[interface{}]interface{}{
				"from": "postgres-database",
			},
		},
		"properties": map[interface{}]interface{}{
			"release_level_backup": true,
		},
	}
	AddOpDefinition(&ops, "replace", path, value)
	return ops
}

func Define_bbr_no_link_ops() []OpDefinition {
	var ops []OpDefinition
	var value interface{}
	var path string

	p := "/instance_groups/name=backup"
	ops = append(ops, OpDefinition{Type: "remove", Path: &p})

	path = "/instance_groups/name=postgres/jobs/name=postgres/provides"
	value = map[interface{}]interface{}{
		"postgres": "nil",
	}
	AddOpDefinition(&ops, "replace", path, value)

	path = "/instance_groups/name=postgres/jobs/-"
	value = map[interface{}]interface{}{
		"name":    "bbr-postgres-db",
		"release": "postgres",
		"properties": map[interface{}]interface{}{
			"release_level_backup": true,
			"postgres": map[interface{}]interface{}{
				"port": "5524",
				"databases": []map[interface{}]interface{}{
					{"name": "sandbox"},
					{"name": "sandbox-2"},
				},
			},
		},
	}
	AddOpDefinition(&ops, "replace", path, value)
	return ops
}

func Define_bbr_not_colocated_ops() []OpDefinition {
	var ops []OpDefinition
	var value interface{}
	var path string

	path = "/instance_groups/name=backup/instances"
	value = 1
	AddOpDefinition(&ops, "replace", path, value)
	return ops
}

func Define_bbr_ssl_verify_full() []OpDefinition {
	var ops []OpDefinition

	ops = Define_bbr_not_colocated_ops()
	ops = append(ops, Define_ssl_ops()...)

	return ops
}

func Define_bbr_ssl_verify_ca() []OpDefinition {
	var ops []OpDefinition
	var value interface{}
	var path string

	ops = Define_bbr_ssl_verify_full()

	path = "/instance_groups/name=backup/jobs/name=bbr-postgres-db/properties/postgres?/ssl_verify_hostname"
	value = false
	AddOpDefinition(&ops, "replace", path, value)

	return ops
}

func Define_bbr_client_certs() []OpDefinition {
	var ops []OpDefinition
	var value interface{}
	var path string

	bbruser := "bbruser"

	ops = Define_bbr_not_colocated_ops()
	ops = append(ops, Define_mutual_ssl_ops()...)

	path = "/instance_groups/name=backup/jobs/name=bbr-postgres-db/properties/postgres?/client_certificate"
	value = "((bbr_user_certs.certificate))"
	AddOpDefinition(&ops, "replace", path, value)

	path = "/instance_groups/name=backup/jobs/name=bbr-postgres-db/properties/postgres?/client_certificate_key"
	value = "((bbr_user_certs.private_key))"
	AddOpDefinition(&ops, "replace", path, value)

	path = "/variables?/name=bbr_user_certs?"
	value = map[interface{}]interface{}{
		"name": "bbr_user_certs",
		"type": "certificate",
		"options": map[interface{}]interface{}{
			"ca":                 "postgres_ca",
			"common_name":        bbruser,
			"alternative_names":  []interface{}{},
			"extended_key_usage": []interface{}{"server_auth", "client_auth"},
		},
	}
	AddOpDefinition(&ops, "replace", path, value)

	path = "/instance_groups/name=postgres/jobs/name=postgres/properties/databases/roles?/name=bbruser?"
	value = map[interface{}]interface{}{
		"name":        bbruser,
		"permissions": []interface{}{"SUPERUSER"},
	}
	AddOpDefinition(&ops, "replace", path, value)

	path = "/instance_groups/name=backup/jobs/name=bbr-postgres-db/properties/postgres?/dbuser"
	value = bbruser
	AddOpDefinition(&ops, "replace", path, value)

	return ops
}

func Define_upgrade_no_copy_ops() []OpDefinition {
	var ops []OpDefinition
	var value interface{}
	var path string

	path = "/instance_groups/name=postgres/jobs/name=postgres/properties/databases/skip_data_copy_in_minor?"
	value = true
	AddOpDefinition(&ops, "replace", path, value)
	return ops
}

func Define_ssl_ops() []OpDefinition {
	var ops []OpDefinition
	var value interface{}
	var path string

	path = "/instance_groups/name=postgres/jobs/name=postgres/properties/databases/tls?/certificate"
	value = "((postgres_cert.certificate))"
	AddOpDefinition(&ops, "replace", path, value)

	path = "/instance_groups/name=postgres/jobs/name=postgres/properties/databases/tls?/private_key"
	value = "((postgres_cert.private_key))"
	AddOpDefinition(&ops, "replace", path, value)

	path = "/variables?/name=postgres_ca?"
	value = map[interface{}]interface{}{
		"name": "postgres_ca",
		"type": "certificate",
		"options": map[interface{}]interface{}{
			"is_ca":       true,
			"common_name": "postgres_ca",
		},
	}
	AddOpDefinition(&ops, "replace", path, value)

	path = "/variables?/name=postgres_cert?"
	value = map[interface{}]interface{}{
		"name": "postgres_cert",
		"type": "certificate",
		"options": map[interface{}]interface{}{
			"ca":                 "postgres_ca",
			"common_name":        "((postgres_host))",
			"alternative_names":  []interface{}{"((postgres_host))", "((postgres_dns))"},
			"extended_key_usage": []interface{}{"server_auth"},
		},
	}
	AddOpDefinition(&ops, "replace", path, value)

	path = "/variables?/name=((certs_matching_certs))?"
	value = map[interface{}]interface{}{
		"name": "((certs_matching_certs))",
		"type": "certificate",
		"options": map[interface{}]interface{}{
			"ca":                 "postgres_ca",
			"common_name":        "((certs_matching_name))",
			"alternative_names":  []interface{}{},
			"extended_key_usage": []interface{}{"server_auth", "client_auth"},
		},
	}
	AddOpDefinition(&ops, "replace", path, value)

	path = "/variables?/name=((certs_mapped_certs))?"
	value = map[interface{}]interface{}{
		"name": "((certs_mapped_certs))",
		"type": "certificate",
		"options": map[interface{}]interface{}{
			"ca":                 "postgres_ca",
			"common_name":        "((certs_mapped_cn))",
			"alternative_names":  []interface{}{},
			"extended_key_usage": []interface{}{"server_auth", "client_auth"},
		},
	}
	AddOpDefinition(&ops, "replace", path, value)

	path = "/variables?/name=((certs_wrong_certs))?"
	value = map[interface{}]interface{}{
		"name": "((certs_wrong_certs))",
		"type": "certificate",
		"options": map[interface{}]interface{}{
			"ca":                 "postgres_ca",
			"common_name":        "((certs_wrong_cn))",
			"alternative_names":  []interface{}{},
			"extended_key_usage": []interface{}{"server_auth", "client_auth"},
		},
	}
	AddOpDefinition(&ops, "replace", path, value)

	path = "/variables?/name=((certs_bad_ca))?"
	value = map[interface{}]interface{}{
		"name": "((certs_bad_ca))",
		"type": "certificate",
		"options": map[interface{}]interface{}{
			"is_ca":       true,
			"common_name": "bad_ca",
		},
	}
	AddOpDefinition(&ops, "replace", path, value)

	return ops
}

func Define_mutual_ssl_ops() []OpDefinition {
	var value interface{}
	var path string

	ops := Define_ssl_ops()

	path = "/instance_groups/name=postgres/jobs/name=postgres/properties/databases/tls?/ca"
	value = "((postgres_cert.ca))"
	AddOpDefinition(&ops, "replace", path, value)

	path = "/instance_groups/name=postgres/jobs/name=postgres/properties/databases/roles?/name=aaa?"
	value = map[interface{}]interface{}{
		"name":        "aaa",
		"common_name": "aaa_2",
	}
	AddOpDefinition(&ops, "replace", path, value)

	path = "/instance_groups/name=postgres/jobs/name=postgres/properties/databases/roles?/name=((certs_matching_name))?"
	value = map[interface{}]interface{}{
		"name": "((certs_matching_name))",
	}
	AddOpDefinition(&ops, "replace", path, value)

	path = "/instance_groups/name=postgres/jobs/name=postgres/properties/databases/roles?/name=((certs_mapped_name))?"
	value = map[interface{}]interface{}{
		"name":        "((certs_mapped_name))",
		"common_name": "((certs_mapped_cn))",
	}
	AddOpDefinition(&ops, "replace", path, value)

	return ops
}

func Define_add_bad_role() []OpDefinition {
	var ops []OpDefinition
	var value interface{}
	var path string

	path = "/instance_groups/name=postgres/jobs/name=postgres/properties/databases/roles?/-"
	var permissions []string
	permissions = append(permissions, "DOESNOTEXIST")
	value = map[interface{}]interface{}{
		"name":        "foo",
		"password":    "foo",
		"permissions": permissions,
	}
	AddOpDefinition(&ops, "replace", path, value)

	return ops
}

func DefineHooks(hooks_timeout string, pre_start string, post_start string, pre_stop string, post_stop string) []OpDefinition {
	var ops []OpDefinition
	var path string

	path = "/instance_groups/name=postgres/jobs/name=postgres/properties/databases/hooks?/timeout?"
	AddOpDefinition(&ops, "replace", path, hooks_timeout)

	path = "/instance_groups/name=postgres/jobs/name=postgres/properties/databases/hooks?/pre_start?"
	AddOpDefinition(&ops, "replace", path, pre_start)

	path = "/instance_groups/name=postgres/jobs/name=postgres/properties/databases/hooks?/post_start?"
	AddOpDefinition(&ops, "replace", path, post_start)

	path = "/instance_groups/name=postgres/jobs/name=postgres/properties/databases/hooks?/pre_stop?"
	AddOpDefinition(&ops, "replace", path, pre_stop)

	path = "/instance_groups/name=postgres/jobs/name=postgres/properties/databases/hooks?/post_stop?"
	AddOpDefinition(&ops, "replace", path, post_stop)

	return ops
}

func (j Janitor) GetOpDefinitions() []OpDefinition {
	var ops []OpDefinition
	var path string

	path = "/instance_groups/name=postgres/jobs/name=postgres/properties/janitor?/timeout?"
	AddOpDefinition(&ops, "replace", path, j.Timeout)

	path = "/instance_groups/name=postgres/jobs/name=postgres/properties/janitor?/interval?"
	AddOpDefinition(&ops, "replace", path, j.Interval)

	path = "/instance_groups/name=postgres/jobs/name=postgres/properties/janitor?/script?"
	AddOpDefinition(&ops, "replace", path, j.Script)

	return ops
}
