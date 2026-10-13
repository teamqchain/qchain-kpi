package ledger

// QChain's chaincode calls (qchain-network/chaincode/QChaincode.js @ 9026aba):
// the arguments, the keys each reads and writes, and its response. Lengths are
// passed in; cmd/kpi-prim produces them with QChain's real code. Present keys
// are read at version (height, 0).

// Credential responses are JSON.stringify({success: true, credentialID}).
func credentialResponse(credKey int) int { return len(`{"success":true,"credentialID":""}`) + credKey }

// IssueCall is issueCredential(holderID, credentialType, issuedAt, expiryDate,
// issuerOrgID, ipfsCID, fieldHashesJSON, credentialHash, issuerSignature,
// issuerPublicKey): reads the holder and checks that the new credential ID is
// free, writes the credential record.
func IssueCall(args []int, credKey, record int, height uint64) Call {
	return Call{Fn: "issueCredential", Args: args,
		Reads:       []Read{{Key: args[0], Exists: true, Block: height}, {Key: credKey}},
		Writes:      []Write{{Key: credKey, Value: record}},
		RespPayload: credentialResponse(credKey)}
}

// UpdateExpiryCall is updateExpiry(credID, expiryDate, credentialHash,
// issuerSignature, issuerPublicKey): rewrites the whole record.
func UpdateExpiryCall(args []int, record int, height uint64) Call {
	return credentialChange("updateExpiry", args, record, height)
}

// SuspendCall is suspendCredential(credID, reason).
func SuspendCall(credKey, reason, record int, height uint64) Call {
	return credentialChange("suspendCredential", []int{credKey, reason}, record, height)
}

// RestoreCall is restoreCredential(credID).
func RestoreCall(credKey, record int, height uint64) Call {
	return credentialChange("restoreCredential", []int{credKey}, record, height)
}

// RevokeCall is revokeCredential(credID).
func RevokeCall(credKey, record int, height uint64) Call {
	return credentialChange("revokeCredential", []int{credKey}, record, height)
}

func credentialChange(fn string, args []int, record int, height uint64) Call {
	k := args[0]
	return Call{Fn: fn, Args: args, Reads: []Read{{Key: k, Exists: true, Block: height}},
		Writes: []Write{{Key: k, Value: record}}, RespPayload: credentialResponse(k)}
}

// RegisterCall is registerHolder(holderID, firstName, lastName): checks that
// the holder is new, writes the holder record and returns it.
func RegisterCall(holderID, first, last, record int) Call {
	return Call{Fn: "registerHolder", Args: []int{holderID, first, last},
		Reads: []Read{{Key: holderID}}, Writes: []Write{{Key: holderID, Value: record}}, RespPayload: record}
}

// BindCall is bindHolderKeys(holderID, kemPublicKey, dsaPublicKey): reads the
// holder, rewrites it with both keys, returns {success, holderID}.
func BindCall(holderID, kemKey, dsaKey, record int, height uint64) Call {
	return Call{Fn: "bindHolderKeys", Args: []int{holderID, kemKey, dsaKey},
		Reads: []Read{{Key: holderID, Exists: true, Block: height}}, Writes: []Write{{Key: holderID, Value: record}},
		RespPayload: len(`{"success":true,"holderID":""}`) + holderID}
}
