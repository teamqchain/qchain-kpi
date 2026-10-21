#!/bin/bash
# E14: creates mychannel and joins the orderer and both peers, as QChain's
# docs/setup.md "Create and Join Channel (Using OSNAdmin)" does. Runs inside a
# hyperledger/fabric-tools container on the network, with the material volume
# at /kpi; addresses are the containers' names instead of localhost.
set -euo pipefail

NETWORK_ROOT=/kpi
CHANNEL_BLOCK=$NETWORK_ROOT/channel-artifacts/mychannel.block
ORDERER_TLS=$NETWORK_ROOT/crypto-material/ordererOrganizations/orderer.example.com/orderers/orderer0.orderer.example.com/tls

# 1. Channel genesis block
export FABRIC_CFG_PATH=$NETWORK_ROOT/config
mkdir -p "$NETWORK_ROOT/channel-artifacts"
configtxgen -profile TwoOrgsChannel -channelID mychannel -outputBlock "$CHANNEL_BLOCK"

# 2. Join the orderer to the channel
export ORDERER_ADMIN_TLS_SIGNCLIENTCERT=$ORDERER_TLS/server.crt
export ORDERER_ADMIN_TLS_PRIVATEKEY=$ORDERER_TLS/server.key
export ORDERER_CA=$ORDERER_TLS/ca.crt
osnadmin channel join --channelID mychannel \
  --config-block "$CHANNEL_BLOCK" \
  -o orderer0.orderer.example.com:7053 \
  --ca-file "$ORDERER_CA" \
  --client-cert "$ORDERER_ADMIN_TLS_SIGNCLIENTCERT" \
  --client-key "$ORDERER_ADMIN_TLS_PRIVATEKEY"

# The single Raft consenter elects itself; wait until the channel is active.
for i in $(seq 1 60); do
  if osnadmin channel list --channelID mychannel -o orderer0.orderer.example.com:7053 \
       --ca-file "$ORDERER_CA" --client-cert "$ORDERER_ADMIN_TLS_SIGNCLIENTCERT" \
       --client-key "$ORDERER_ADMIN_TLS_PRIVATEKEY" 2>/dev/null | grep -q '"status": "active"'; then
    break
  fi
  [ "$i" -lt 60 ] || { echo "channel.sh: the orderer did not activate mychannel" >&2; exit 1; }
  sleep 1
done

# 3. Join both peers
# shellcheck source=/dev/null
source "$NETWORK_ROOT/scripts/env-gov.sh"
peer channel join -b "$CHANNEL_BLOCK"
# shellcheck source=/dev/null
source "$NETWORK_ROOT/scripts/env-gen.sh"
peer channel join -b "$CHANNEL_BLOCK"
echo "channel.sh: mychannel created; orderer and both peers joined"
