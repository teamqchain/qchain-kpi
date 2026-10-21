'use strict';

// Unit tests for the E14 synthetic chaincode against an in-memory ledger
// (Node's built-in test runner, as QChain's chaincode tests):
//   npm install && npm test

const test = require('node:test');
const assert = require('node:assert/strict');
const KpiPayload = require('../KpiPayload');

class MockStub {
    constructor() {
        this.state = new Map();
    }
    async getState(key) {
        return this.state.has(key) ? Buffer.from(this.state.get(key)) : Buffer.alloc(0);
    }
    async putState(key, value) {
        this.state.set(key, value.toString());
    }
}

function makeCtx(stub, role = 'issuer') {
    return {
        stub,
        clientIdentity: {
            assertAttributeValue: (name, value) => name === 'role' && value === role,
            getMSPID: () => 'GeneralMSP',
            getID: () => 'x509::/OU=client/CN=issuer1::/CN=ca',
        },
    };
}

async function seeded() {
    const cc = new KpiPayload();
    const stub = new MockStub();
    const ctx = makeCtx(stub);
    await cc.seedHolder(ctx);
    return { cc, stub, ctx };
}

test('putPayload stores the payload in a sorted-key record and returns a short result', async () => {
    const { cc, stub, ctx } = await seeded();
    const payload = 'ab'.repeat(2420);
    const out = JSON.parse(await cc.putPayload(ctx, 'P-0001', payload));
    assert.deepEqual(out, { success: true, id: 'P-0001' });
    const raw = stub.state.get('P-0001');
    const rec = JSON.parse(raw);
    assert.equal(rec.Payload, payload);
    assert.equal(rec.IssuerOrgID, 'GeneralMSP');
    assert.deepEqual(Object.keys(rec), [...Object.keys(rec)].sort(), 'keys are sorted');
    assert.equal(JSON.parse(await cc.getPayload(ctx, 'P-0001')).Payload, payload);
});

test('putPayload refuses a repeated id, a bad payload, a missing role and a missing holder', async () => {
    const { cc, stub, ctx } = await seeded();
    await cc.putPayload(ctx, 'P-1', '00');
    await assert.rejects(cc.putPayload(ctx, 'P-1', '00'), /already exists/);
    await assert.rejects(cc.putPayload(ctx, 'P-2', 'XY'), /lowercase hex/);
    await assert.rejects(cc.putPayload(ctx, 'P-3', 'abc'), /even length/);
    await assert.rejects(cc.putPayload(ctx, 'bad id', '00'), /id must be/);
    await assert.rejects(cc.putPayload(makeCtx(stub, 'verifier'), 'P-4', '00'), /Access denied/);
    await assert.rejects(new KpiPayload().putPayload(makeCtx(new MockStub()), 'P-5', '00'), /Holder not found/);
});

test('seedHolder runs once', async () => {
    const { cc, ctx } = await seeded();
    await assert.rejects(cc.seedHolder(ctx), /already registered/);
});

test('a 206 KB payload round-trips', async () => {
    const { cc, ctx } = await seeded();
    const payload = '5a'.repeat(103000);
    await cc.putPayload(ctx, 'P-big', payload);
    assert.equal(JSON.parse(await cc.getPayload(ctx, 'P-big')).Payload.length, 206000);
});
