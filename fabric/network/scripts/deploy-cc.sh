#!/bin/bash
# E14: deploys the synthetic chaincode kpipayload (version 1.0, sequence 1) with
# the Fabric v2 lifecycle, as QChain's docs/setup.md "Deploy Chaincode" does for
# qchaincode: package once, install and approve on each organisation, commit with
# both peers. Then seeds the holder record putPayload reads, as the issuer.
# Runs inside a hyperledger/fabric-tools container on the network, with the
# material volume at /kpi. The peers build the Node chaincode with
# hyperledger/fabric-nodeenv (npm install needs the internet).
set -euo pipefail

NETWORK_ROOT=/kpi
ORDERER_CA=$NETWORK_ROOT/crypto-material/ordererOrganizations/orderer.example.com/orderers/orderer0.orderer.example.com/tls/ca.crt
GOV_TLS=$NETWORK_ROOT/crypto-material/peerOrganizations/government.uae.com/peers/peer0.government.uae.com/tls/ca.crt
GEN_TLS=$NETWORK_ROOT/crypto-material/peerOrganizations/general.uae.com/peers/peer0.general.uae.com/tls/ca.crt
PKG=$NETWORK_ROOT/packages/kpipayload_1.0.tar.gz
ORDERER=(-o orderer0.orderer.example.com:7050 --ordererTLSHostnameOverride orderer0.orderer.example.com --tls --cafile "$ORDERER_CA")
BOTH=(--peerAddresses peer0.government.uae.com:7051 --tlsRootCertFiles "$GOV_TLS"
      --peerAddresses peer0.general.uae.com:9051 --tlsRootCertFiles "$GEN_TLS")

# 1. Package
mkdir -p "$NETWORK_ROOT/packages"
# shellcheck source=/dev/null
source "$NETWORK_ROOT/scripts/env-gov.sh"
peer lifecycle chaincode package "$PKG" --path "$NETWORK_ROOT/chaincode" --lang node --label kpipayload_1.0
CC_PACKAGE_ID=$(peer lifecycle chaincode calculatepackageid "$PKG")
echo "deploy-cc.sh: package id $CC_PACKAGE_ID"

# 2. Install and approve on Government, 3. on General
for org in gov gen; do
  # shellcheck source=/dev/null
  source "$NETWORK_ROOT/scripts/env-$org.sh"
  peer lifecycle chaincode install "$PKG"
  peer lifecycle chaincode approveformyorg "${ORDERER[@]}" --channelID mychannel --name kpipayload \
    --version 1.0 --package-id "$CC_PACKAGE_ID" --sequence 1
done

# 4. Commit (sequence 1)
peer lifecycle chaincode checkcommitreadiness --channelID mychannel --name kpipayload --version 1.0 --sequence 1 --output json
peer lifecycle chaincode commit "${ORDERER[@]}" --channelID mychannel --name kpipayload --version 1.0 --sequence 1 "${BOTH[@]}"
peer lifecycle chaincode querycommitted --channelID mychannel --name kpipayload

# 5. Seed the holder record, as QChain's issuer (role=issuer), endorsed by both peers.
export CORE_PEER_MSPCONFIGPATH=$NETWORK_ROOT/crypto-material/peerOrganizations/general.uae.com/users/issuer1@general.uae.com/msp
peer chaincode invoke "${ORDERER[@]}" -C mychannel -n kpipayload "${BOTH[@]}" --waitForEvent \
  -c '{"Args":["seedHolder"]}'
echo "deploy-cc.sh: kpipayload 1.0 committed and seeded"
