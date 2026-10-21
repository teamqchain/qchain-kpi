'use strict';

// E14 synthetic chaincode (QChain KPI study). It does on the ledger what
// QChaincode v2.0's issueCredential does, without its business fields, so that
// a transaction's cost depends on its size the way QChain's does:
//   - the caller must carry the attribute role=issuer (QChain's _checkAccess);
//   - one read of the holder record (QChain's _getHolder) and one existence
//     check of the new key (QChain's _exists);
//   - one write: a JSON record with sorted keys (QChain's _put, the same two
//     libraries), which holds the payload, as QChain's record holds the
//     signature and the public keys;
//   - a short JSON result, as issueCredential returns.
// The payload is hex text, as QChain stores keys and signatures. There is no
// cryptography here (QChain's chaincode does none either).

const stringify = require('json-stringify-deterministic');
const sortKeysRecursive = require('sort-keys-recursive');
const { Contract } = require('fabric-contract-api');

const HOLDER_ID = 'H-KPI';
const HEX = /^[0-9a-f]+$/;
const ID = /^[A-Za-z0-9._-]{1,256}$/;

class KpiPayload extends Contract {
    async init(ctx) {
        return JSON.stringify('Initialization successful.');
    }

    // Writes the holder record putPayload reads (once, at deployment).
    async seedHolder(ctx) {
        this._checkAccess(ctx, 'issuer');
        if (await this._exists(ctx, HOLDER_ID)) {
            throw new Error(`Holder already registered: ${HOLDER_ID}`);
        }
        await this._put(ctx, HOLDER_ID, { DocType: 'holder', ID: HOLDER_ID, FirstName: 'KPI', LastName: 'Holder' });
        return JSON.stringify({ success: true, holderID: HOLDER_ID });
    }

    // Stores payloadHex under id. The client chooses id (unique per transaction).
    async putPayload(ctx, id, payloadHex) {
        this._checkAccess(ctx, 'issuer');
        const holder = await ctx.stub.getState(HOLDER_ID);
        if (!holder || holder.length === 0) {
            throw new Error(`Holder not found: ${HOLDER_ID}`);
        }
        if (typeof id !== 'string' || !ID.test(id)) {
            throw new Error('id must be 1 to 256 characters of [A-Za-z0-9._-]');
        }
        if (typeof payloadHex !== 'string' || payloadHex.length % 2 !== 0 || !HEX.test(payloadHex)) {
            throw new Error('payload must be lowercase hex with an even length');
        }
        if (await this._exists(ctx, id)) {
            throw new Error(`Payload already exists: ${id}`);
        }
        await this._put(ctx, id, {
            DocType: 'kpi-payload',
            ID: id,
            Holder: HOLDER_ID,
            Issuer: ctx.clientIdentity.getID(),
            IssuerOrgID: ctx.clientIdentity.getMSPID(),
            Payload: payloadHex,
        });
        return JSON.stringify({ success: true, id });
    }

    async getPayload(ctx, id) {
        const data = await ctx.stub.getState(id);
        if (!data || data.length === 0) {
            throw new Error(`Payload not found: ${id}`);
        }
        return data.toString('utf8');
    }

    _checkAccess(ctx, role) {
        if (!ctx.clientIdentity.assertAttributeValue('role', role)) {
            throw new Error(`Access denied. Only a ${role} official may access this function.`);
        }
    }

    async _put(ctx, key, value) {
        await ctx.stub.putState(key, Buffer.from(stringify(sortKeysRecursive(value))));
    }

    async _exists(ctx, key) {
        const data = await ctx.stub.getState(key);
        return !!data && data.length > 0;
    }
}

module.exports = KpiPayload;
