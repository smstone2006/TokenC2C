#!/bin/sh
set -eu

docker_bin=/share/CACHEDEV1_DATA/.qpkg/container-station/bin/docker
deploy_dir=${SUBC2C_DEPLOY_DIR:-/share/CACHEDEV1_DATA/Container/subc2c/deploy}
compose_file="$deploy_dir/docker-compose.yml"
env_file="$deploy_dir/.env"
qnet_name=${SUBC2C_QNET_NAME:-qnet-static-eth2-b0591c}
app_container=subc2c-nas-app

set -a
. "$env_file"
set +a

: "${SUBC2C_BRIDGE_IP:?SUBC2C_BRIDGE_IP is required}"

"$docker_bin" compose --env-file "$env_file" -f "$compose_file" config -q
"$docker_bin" compose --env-file "$env_file" -f "$compose_file" up -d

i=0
while [ "$i" -lt 90 ]; do
  health=$("$docker_bin" inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$app_container")
  [ "$health" = healthy ] && break
  i=$((i + 1))
  sleep 1
done
[ "${health:-none}" = healthy ]

if "$docker_bin" inspect --format '{{json .NetworkSettings.Networks}}' "$app_container" | grep -q "\"$qnet_name\""; then
  "$docker_bin" network disconnect "$qnet_name" "$app_container"
fi
"$docker_bin" network connect --ip "$SUBC2C_BRIDGE_IP" "$qnet_name" "$app_container"
curl -fsS --connect-timeout 5 --max-time 10 "http://$SUBC2C_BRIDGE_IP:8080/health" >/dev/null

echo "TokenC2C is healthy at $SUBC2C_BRIDGE_IP:8080"
