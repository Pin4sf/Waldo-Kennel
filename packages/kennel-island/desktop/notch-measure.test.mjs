import assert from "node:assert/strict";
import test from "node:test";
import { measureNotch, notchHelperPath, parseMeasurement } from "./notch-measure.mjs";

function execFileReturning(result) {
	return (_command, _args, _options, callback) => callback(result.error ?? null, result.stdout ?? "");
}

test("the helper path lives beside the rest of the desktop code", () => {
	assert.equal(notchHelperPath("/Apps/Island.app"), "/Apps/Island.app/desktop/helpers/kennel-notch");
});

test("a measured notch is read as points", () => {
	assert.deepEqual(
		parseMeasurement('{"hasNotch":true,"notchWidth":220,"notchHeight":38,"menuBarHeight":38}'),
		{ hasNotch: true, notchWidth: 220, notchHeight: 38 },
	);
});

test("a measured flat bezel is an answer, not a failure", () => {
	assert.deepEqual(parseMeasurement('{"hasNotch":false,"menuBarHeight":25}'), { hasNotch: false });
});

test("a half-measurement is refused, because falling back beats guessing", () => {
	assert.equal(parseMeasurement('{"hasNotch":true,"notchWidth":220}'), null);
	assert.equal(parseMeasurement('{"hasNotch":true,"notchHeight":38}'), null);
	assert.equal(parseMeasurement('{"hasNotch":true,"notchWidth":0,"notchHeight":38}'), null);
});

test("anything that is not a measurement is null", () => {
	for (const stdout of ["", "not json", "[]", "null", '"220"', "{}", '{"hasNotch":"yes"}']) {
		assert.equal(parseMeasurement(stdout), null, stdout);
	}
});

test("a helper that answers is trusted", async () => {
	const measured = await measureNotch({
		helperPath: "/helper",
		execFile: execFileReturning({ stdout: '{"hasNotch":true,"notchWidth":204,"notchHeight":37}' }),
	});

	assert.deepEqual(measured, { hasNotch: true, notchWidth: 204, notchHeight: 37 });
});

test("a helper that fails, is missing, or throws leaves the caller deriving", async () => {
	assert.equal(
		await measureNotch({ helperPath: "/helper", execFile: execFileReturning({ error: new Error("ENOENT") }) }),
		null,
	);
	assert.equal(
		await measureNotch({ helperPath: "/helper", execFile: execFileReturning({ stdout: "boom" }) }),
		null,
	);
	assert.equal(
		await measureNotch({
			helperPath: "/helper",
			execFile: () => {
				throw new Error("spawn failed");
			},
		}),
		null,
	);
});

test("the measurement is bounded in time, because it runs before the first window", async () => {
	let seenTimeout;
	await measureNotch({
		helperPath: "/helper",
		timeoutMs: 500,
		execFile: (_command, _args, options, callback) => {
			seenTimeout = options.timeout;
			callback(null, '{"hasNotch":false}');
		},
	});

	assert.equal(seenTimeout, 500);
});
