#!/bin/bash -eu

preflight_check() {
  set +x
  test -n "${BOSH_DIRECTOR_IP}"
  test -n "${CONFIG_FILE_NAME}"
  test -n "${API_USER}"
  test -n "${API_PASSWORD}"
  set -x
}

function main(){
  local root="${1}"
  preflight_check

  local SYSTEM_DOMAIN=${BOSH_DIRECTOR_IP}.nip.io
  local CONFIG_FILE="${root}/cats-config/${CONFIG_FILE_NAME}"

  cat <<EOF > "${CONFIG_FILE}"
{
  "api": "api.${SYSTEM_DOMAIN}",
  "apps_domain": "${SYSTEM_DOMAIN}",
  "admin_user": "${API_USER}",
  "admin_password": "${API_PASSWORD}",
  "skip_ssl_validation": true,
  "use_http": true,
  "include_apps": true,
  "timeout_scale": 5,
  "include_backend_compatibility": false,
  "include_capi_experimental": false,
  "include_capi_no_bridge": false,
  "include_container_networking": false,
  "include_credhub" : false,
  "include_detect": false,
  "include_docker": false,
  "include_internet_dependent": false,
  "include_isolation_segments": false,
  "include_persistent_app": false,
  "include_private_docker_registry": false,
  "include_privileged_container_support": false,
  "include_route_services": false,
  "include_routing": false,
  "include_routing_isolation_segments": false,
  "include_security_groups": false,
  "include_services": false,
  "include_service_instance_sharing": false,
  "include_ssh": false,
  "include_sso": false,
  "include_tasks": false,
  "include_v3": false,
  "include_zipkin": false
}
EOF

}

main "${PWD}"
