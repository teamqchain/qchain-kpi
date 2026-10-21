#!/bin/bash
# E14: QChain's Fabric network, rebuilt on the Mac inside Colima (docs/e14-plan.md).
#
#   scripts/e14-network.sh setup     # first time (needs the internet): everything below, in order
#   scripts/e14-network.sh start     # start the stopped containers again (after a Mac sleep)
#   scripts/e14-network.sh stop      # stop the containers; ledger and certificates are kept
#   scripts/e14-network.sh status    # containers, memory use, channel height
#
# It mirrors qchain-network/docker/docker-compose{,-ca}.yaml (QChain commit
# 71a1db6) with plain `docker run` (the Mac has no docker compose): the same
# container names, environment variables and mount points, the same images
# pinned (Fabric 2.5.15, Fabric CA 1.5.15, CouchDB 3.3.2), one CouchDB shared by
# both peers (as live), and the same certificates, channel and lifecycle steps
# (QChain's registerEnroll.sh, env-gov.sh and env-gen.sh run unchanged).
# Differences, all outside the measured path: no host ports are published (the
# E14 client runs inside the same Docker network), and files live in the
# named volume kpi-fabric-material (Colima has no host mounts), mounted with
# volume sub-paths at the live paths.
#
# Every step is skipped when it has been done before, so a failed setup can be
# run again without deleting anything. Only containers, volumes and the network
# labelled org.qchain.kpi.e14=1 are touched.
set -euo pipefail

repo="$(cd "$(dirname "$0")/.." && pwd -P)"
NET=docker_fabric_net   # the live name (compose project "docker"); the peers' CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE
VOL=kpi-fabric-material
IMG_PEER=hyperledger/fabric-peer:2.5.15
IMG_ORDERER=hyperledger/fabric-orderer:2.5.15
IMG_TOOLS=hyperledger/fabric-tools:2.5.15
IMG_CA=hyperledger/fabric-ca:1.5.15
IMG_NODEENV=hyperledger/fabric-nodeenv:2.5
IMG_COUCH=couchdb:3.3.2
LBL=(--label org.qchain.kpi=1 --label org.qchain.kpi.e14=1)
CONTAINERS=(ca.government.uae.com ca.general.uae.com ca.orderer.example.com couchdb0
            orderer0.orderer.example.com peer0.government.uae.com peer0.general.uae.com)

die() { echo "e14-network.sh: $*" >&2; exit 1; }
note() { echo "e14-network.sh: $*" >&2; }

# sub VOLUME SUBPATH TARGET: a --mount of part of a named volume.
sub() { printf 'type=volume,src=%s,dst=%s,volume-subpath=%s' "$1" "$3" "$2"; }

tools() { docker run --rm --network "$NET" --mount "type=volume,src=$VOL,dst=/kpi" "${LBL[@]}" --entrypoint bash "$IMG_TOOLS" "$@"; }
done_marker() { tools -c "test -f /kpi/state/$1" >/dev/null 2>&1; }
mark() { tools -c "mkdir -p /kpi/state && date -u +%FT%TZ > /kpi/state/$1"; }

ours() { [ "$(docker inspect -f '{{index .Config.Labels "org.qchain.kpi.e14"}}' "$1" 2>/dev/null)" = "1" ]; }
exists() { docker inspect "$1" >/dev/null 2>&1; }
running() { [ "$(docker inspect -f '{{.State.Running}}' "$1" 2>/dev/null)" = "true" ]; }
healthy() { [ "$(docker inspect -f '{{.State.Health.Status}}' "$1" 2>/dev/null)" = "healthy" ]; }

# run_once NAME DOCKER-RUN-ARGS...: creates and starts the container unless it
# exists (then it is started if stopped). A container of that name that is not
# ours stops the script.
run_once() {
  local name="$1"; shift
  if exists "$name"; then
    ours "$name" || die "a container named $name exists and is not E14's; nothing was changed"
    running "$name" || docker start "$name" >/dev/null
    return 0
  fi
  docker run -d --name "$name" --network "$NET" "${LBL[@]}" "$@" >/dev/null
}

wait_for() { # description seconds command...
  local what="$1" secs="$2" i; shift 2
  for ((i = 0; i < secs; i++)); do
    if "$@" >/dev/null 2>&1; then return 0; fi
    sleep 1
  done
  die "timed out after ${secs}s waiting for $what"
}

check_images() {
  local i missing=()
  for i in "$IMG_PEER" "$IMG_ORDERER" "$IMG_TOOLS" "$IMG_CA" "$IMG_NODEENV" "$IMG_COUCH"; do
    docker image inspect "$i" >/dev/null 2>&1 || missing+=("$i")
  done
  [ "${#missing[@]}" -eq 0 ] || die "images not pulled: ${missing[*]} (docker pull them first)"
}

# ------------------------------------------------------------------ steps

create_net_and_volumes() {
  local v
  if ! docker network inspect "$NET" >/dev/null 2>&1; then
    docker network create "${LBL[@]}" "$NET" >/dev/null
  fi
  for v in "$VOL" kpi-fabric-orderer0.orderer.example.com kpi-fabric-peer0.government.uae.com kpi-fabric-peer0.general.uae.com; do
    docker volume inspect "$v" >/dev/null 2>&1 || docker volume create "${LBL[@]}" "$v" >/dev/null
  done
}

load_material() { # copies fabric/network and fabric/chaincode into the volume (first time only)
  done_marker loaded && return 0
  local c=kpi-fabric-loader
  exists "$c" && docker rm "$c" >/dev/null
  docker create --name "$c" "${LBL[@]}" --mount "type=volume,src=$VOL,dst=/kpi" --entrypoint true "$IMG_TOOLS" >/dev/null
  # macOS tar adds extended attributes that Docker refuses.
  (cd "$repo/fabric/network" && COPYFILE_DISABLE=1 tar --no-xattrs --no-mac-metadata -cf - --exclude .DS_Store .) | docker cp - "$c:/kpi"
  (cd "$repo/fabric" && COPYFILE_DISABLE=1 tar --no-xattrs --no-mac-metadata -cf - --exclude .DS_Store --exclude node_modules chaincode) | docker cp - "$c:/kpi"
  docker rm "$c" >/dev/null
  tools -c 'chmod +x /kpi/scripts/*.sh'
  mark loaded
}

start_cas() {
  local org caname port ops
  while read -r org caname port ops; do
    run_once "ca.$org" --hostname "ca.$org" \
      -e FABRIC_CA_HOME=/etc/hyperledger/fabric-ca-server -e "FABRIC_CA_SERVER_CA_NAME=$caname" \
      -e FABRIC_CA_SERVER_TLS_ENABLED=true -e "FABRIC_CA_SERVER_PORT=$port" \
      -e "FABRIC_CA_SERVER_OPERATIONS_LISTENADDRESS=0.0.0.0:$ops" \
      --mount "$(sub "$VOL" "fabric-ca/${org%%.*}" /etc/hyperledger/fabric-ca-server)" \
      "$IMG_CA" sh -c 'fabric-ca-server start -b admin:adminpw -d'
  done <<'EOF'
government.uae.com GovernmentCA 7054 17054
general.uae.com GeneralCA 8054 18054
orderer.example.com OrdererCA 9054 19054
EOF
  for org in government general orderer; do
    wait_for "the $org CA" 60 tools -c "test -s /kpi/fabric-ca/$org/ca-cert.pem && test -s /kpi/fabric-ca/$org/tls-cert.pem"
  done
}

enroll() { # QChain's registerEnroll.sh, unchanged, in a Fabric CA client container
  done_marker enrolled && return 0
  docker run --rm --network "$NET" --mount "type=volume,src=$VOL,dst=/kpi" "${LBL[@]}" \
    --entrypoint bash "$IMG_CA" /kpi/scripts/registerEnroll.sh
  mark enrolled
}

M=crypto-material
GOVP=$M/peerOrganizations/government.uae.com/peers/peer0.government.uae.com
GENP=$M/peerOrganizations/general.uae.com/peers/peer0.general.uae.com
ORD=$M/ordererOrganizations/orderer.example.com/orderers/orderer0.orderer.example.com

start_nodes() {
  run_once orderer0.orderer.example.com \
    -e FABRIC_LOGGING_SPEC=INFO -e ORDERER_GENERAL_LISTENADDRESS=0.0.0.0 -e ORDERER_GENERAL_LISTENPORT=7050 \
    -e ORDERER_GENERAL_LOCALMSPID=OrdererMSP -e ORDERER_GENERAL_LOCALMSPDIR=/var/hyperledger/orderer/msp \
    -e ORDERER_GENERAL_TLS_ENABLED=true \
    -e ORDERER_GENERAL_TLS_PRIVATEKEY=/var/hyperledger/orderer/tls/server.key \
    -e ORDERER_GENERAL_TLS_CERTIFICATE=/var/hyperledger/orderer/tls/server.crt \
    -e 'ORDERER_GENERAL_TLS_ROOTCAS=[/var/hyperledger/orderer/tls/ca.crt]' \
    -e ORDERER_GENERAL_CLUSTER_CLIENTCERTIFICATE=/var/hyperledger/orderer/tls/server.crt \
    -e ORDERER_GENERAL_CLUSTER_CLIENTPRIVATEKEY=/var/hyperledger/orderer/tls/server.key \
    -e 'ORDERER_GENERAL_CLUSTER_ROOTCAS=[/var/hyperledger/orderer/tls/ca.crt]' \
    -e ORDERER_GENERAL_BOOTSTRAPMETHOD=none -e ORDERER_CHANNELPARTICIPATION_ENABLED=true \
    -e ORDERER_ADMIN_TLS_ENABLED=true -e ORDERER_ADMIN_LISTENADDRESS=0.0.0.0:7053 \
    -e ORDERER_ADMIN_TLS_CERTIFICATE=/var/hyperledger/orderer/tls/server.crt \
    -e ORDERER_ADMIN_TLS_PRIVATEKEY=/var/hyperledger/orderer/tls/server.key \
    -e 'ORDERER_ADMIN_TLS_ROOTCAS=[/var/hyperledger/orderer/tls/ca.crt]' \
    -e ORDERER_ADMIN_TLS_CLIENTAUTHREQUIRED=true \
    -e 'ORDERER_ADMIN_TLS_CLIENTROOTCAS=[/var/hyperledger/orderer/tls/ca.crt]' \
    --mount "$(sub "$VOL" "$ORD/msp" /var/hyperledger/orderer/msp)" \
    --mount "$(sub "$VOL" "$ORD/tls" /var/hyperledger/orderer/tls)" \
    --mount type=volume,src=kpi-fabric-orderer0.orderer.example.com,dst=/var/hyperledger/production/orderer \
    "$IMG_ORDERER"

  run_once couchdb0 -e COUCHDB_USER=admin -e COUCHDB_PASSWORD=adminpw \
    --health-cmd 'curl -f http://admin:adminpw@localhost:5984/' \
    --health-interval 5s --health-timeout 5s --health-retries 10 "$IMG_COUCH"
  wait_for "CouchDB to be healthy" 120 healthy couchdb0

  local id msp port cc ops dir
  while read -r id msp port cc ops dir; do
    run_once "$id" \
      -e CORE_VM_ENDPOINT=unix:///host/var/run/docker.sock -e "CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE=$NET" \
      -e FABRIC_LOGGING_SPEC=INFO -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_PROFILE_ENABLED=false \
      -e CORE_PEER_TLS_CERT_FILE=/etc/hyperledger/fabric/tls/server.crt \
      -e CORE_PEER_TLS_KEY_FILE=/etc/hyperledger/fabric/tls/server.key \
      -e CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt \
      -e "CORE_PEER_ID=$id" -e "CORE_PEER_ADDRESS=$id:$port" -e "CORE_PEER_LISTENADDRESS=0.0.0.0:$port" \
      -e "CORE_PEER_CHAINCODEADDRESS=$id:$cc" -e "CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:$cc" \
      -e "CORE_PEER_GOSSIP_BOOTSTRAP=$id:$port" -e "CORE_PEER_GOSSIP_EXTERNALENDPOINT=$id:$port" \
      -e "CORE_PEER_LOCALMSPID=$msp" -e CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/msp \
      -e "CORE_OPERATIONS_LISTENADDRESS=0.0.0.0:$ops" \
      -e CORE_LEDGER_STATE_STATEDATABASE=CouchDB -e CORE_LEDGER_STATE_COUCHDBCONFIG_COUCHDBADDRESS=couchdb0:5984 \
      -e CORE_LEDGER_STATE_COUCHDBCONFIG_USERNAME=admin -e CORE_LEDGER_STATE_COUCHDBCONFIG_PASSWORD=adminpw \
      -v /var/run/docker.sock:/host/var/run/docker.sock \
      --mount "$(sub "$VOL" "$dir/msp" /etc/hyperledger/fabric/msp)" \
      --mount "$(sub "$VOL" "$dir/tls" /etc/hyperledger/fabric/tls)" \
      --mount "type=volume,src=kpi-fabric-$id,dst=/var/hyperledger/production" \
      "$IMG_PEER"
  done <<EOF
peer0.government.uae.com GovernmentMSP 7051 7052 9444 $GOVP
peer0.general.uae.com GeneralMSP 9051 9052 9445 $GENP
EOF
  for c in orderer0.orderer.example.com peer0.government.uae.com peer0.general.uae.com; do
    wait_for "$c to start" 60 sh -c "docker logs $c 2>&1 | grep -Eq 'Beginning to serve requests|Started peer with ID|Starting orderer'"
  done
}

channel() {
  done_marker channel && return 0
  tools /kpi/scripts/channel.sh
  mark channel
}

deploy() {
  done_marker chaincode && return 0
  tools /kpi/scripts/deploy-cc.sh
  mark chaincode
}

status() {
  docker ps -a --filter label=org.qchain.kpi.e14=1 --format 'table {{.Names}}\t{{.Status}}\t{{.Image}}'
  docker ps -a --filter name=dev-peer0 --format 'table {{.Names}}\t{{.Status}}'
  # shellcheck disable=SC2046 # one argument per running container
  docker stats --no-stream --format 'table {{.Name}}\t{{.MemUsage}}\t{{.CPUPerc}}' $(docker ps -q) 2>/dev/null || true
}

# ------------------------------------------------------------------- main

cmd="${1:-}"
case "$cmd" in
  setup)
    docker info >/dev/null 2>&1 || die "Docker is not reachable (is Colima running?)"
    check_images
    create_net_and_volumes
    load_material
    start_cas
    enroll
    start_nodes
    channel
    deploy
    note "network ready: mychannel with kpipayload 1.0 (QChain's topology, Fabric 2.5.15)"
    status
    ;;
  start)
    for c in "${CONTAINERS[@]}"; do
      exists "$c" || die "no container $c; run setup first"
      ours "$c" || die "$c is not E14's"
      running "$c" || docker start "$c" >/dev/null
    done
    note "started; the chaincode containers start on the first transaction"
    ;;
  stop)
    for c in peer0.general.uae.com peer0.government.uae.com orderer0.orderer.example.com couchdb0 \
             ca.orderer.example.com ca.general.uae.com ca.government.uae.com; do
      if exists "$c" && ours "$c" && running "$c"; then docker stop "$c" >/dev/null; fi
    done
    for c in $(docker ps -q --filter name=dev-peer0); do docker stop "$c" >/dev/null; done
    note "stopped (ledger, certificates and chaincode are kept)"
    ;;
  status)
    status
    ;;
  *)
    die "usage: e14-network.sh setup|start|stop|status"
    ;;
esac
