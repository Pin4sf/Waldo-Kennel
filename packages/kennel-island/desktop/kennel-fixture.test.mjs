import assert from "node:assert/strict";
import path from "node:path";
import test from "node:test";

import {
	defaultAppPathCandidates,
	firstExistingAppPath,
	handleRequest,
} from "../scripts/kennel-fixture.mjs";

function responseHarness() {
	let status = null;
	let headers = null;
	let body = null;
	return {
		response: {
			writeHead(nextStatus, nextHeaders) {
				status = nextStatus;
				headers = nextHeaders;
			},
			end(payload) {
				body = payload;
			},
		},
		result() {
			return { status, headers, body };
		},
	};
}

test("the fixture health endpoint identifies a live Kennel daemon", async () => {
	const harness = responseHarness();
	await handleRequest({ method: "GET", url: "/healthz" }, harness.response, {});

	const result = harness.result();
	assert.equal(result.status, 200);
	assert.equal(result.headers["cache-control"], "no-store");
	assert.deepEqual(JSON.parse(result.body), {
		status: "ok",
		service: "kennel-daemon",
		pid: process.pid,
	});
});

test("default fixture app discovery starts at the unified repository frontend", async () => {
	const root = path.resolve("/workspace/Waldo-Kennel/packages/kennel-island");
	const expected = path.resolve("/workspace/Waldo-Kennel/frontend/out/Kennel-darwin-arm64/Kennel.app");
	const candidates = defaultAppPathCandidates(root);
	assert.equal(candidates[0], expected);
	assert.equal(
		candidates[1],
		path.resolve("/workspace/Waldo-Kennel/frontend/node_modules/electron/dist/Electron.app"),
	);

	const attempts = [];
	const selected = await firstExistingAppPath({
		env: {},
		root,
		fsModule: {
			async stat(candidate) {
				attempts.push(candidate);
				if (candidate === expected) return { isDirectory: () => true };
				throw Object.assign(new Error("missing"), { code: "ENOENT" });
			},
		},
	});

	assert.equal(selected, expected);
	assert.deepEqual(attempts, [expected]);
});

test("default fixture app discovery retains the former sibling checkout fallback", () => {
	const root = path.resolve("/workspace/kennel-island");
	assert.ok(defaultAppPathCandidates(root).includes(
		path.resolve("/workspace/Waldo-Kennel/frontend/out/Kennel-darwin-arm64/Kennel.app"),
	));
});
