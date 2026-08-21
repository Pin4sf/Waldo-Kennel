import assert from "node:assert/strict";
import test from "node:test";

import {
	appStatePathForRunFile,
	KennelLaunchError,
	resolveKennelAppPath,
	resolveKennelRunFilePath,
} from "./runtime-helpers.mjs";

test("run-file discovery honors only an absolute override and keeps dev state isolated", () => {
	assert.equal(
		resolveKennelRunFilePath({
			env: { AO_RUN_FILE: "/private/tmp/ao/running.json" },
			home: "/Users/tester",
			dev: true,
		}),
		"/private/tmp/ao/running.json",
	);
	assert.equal(
		resolveKennelRunFilePath({ env: {}, home: "/Users/tester", dev: true }),
		"/Users/tester/.ao/dev/running.json",
	);
	assert.equal(
		resolveKennelRunFilePath({ env: {}, home: "/Users/tester", dev: false }),
		"/Users/tester/.ao/running.json",
	);
	assert.equal(
		resolveKennelRunFilePath({
			env: { AO_RUN_FILE: "relative/running.json" },
			home: "/Users/tester",
			dev: false,
		}),
		"/Users/tester/.ao/running.json",
	);
});

test("the Kennel app marker is always resolved beside the selected run file", () => {
	assert.equal(
		appStatePathForRunFile("/Users/tester/.ao/dev/running.json"),
		"/Users/tester/.ao/dev/app-state.json",
	);
	assert.throws(() => appStatePathForRunFile("running.json"), /absolute run file/);
});

test("a macOS app marker resolves only to an existing local application bundle", async () => {
	const calls = [];
	const fs = {
		async readFile(file, encoding) {
			calls.push(["readFile", file, encoding]);
			return JSON.stringify({ schemaVersion: 2, appPath: "/Applications/Kennel.app" });
		},
		async realpath(file) {
			calls.push(["realpath", file]);
			return "/Applications/Kennel.app";
		},
		async stat(file) {
			calls.push(["stat", file]);
			return { isDirectory: () => true, isFile: () => false };
		},
	};

	const appPath = await resolveKennelAppPath({
		fs,
		appStatePath: "/Users/tester/.ao/app-state.json",
		platform: "darwin",
	});

	assert.equal(appPath, "/Applications/Kennel.app");
	assert.deepEqual(calls, [
		["readFile", "/Users/tester/.ao/app-state.json", "utf8"],
		["realpath", "/Applications/Kennel.app"],
		["stat", "/Applications/Kennel.app"],
	]);
});

test("an invalid, stale, or non-app marker returns one safe launch error", async (t) => {
	async function expectUnavailable(fs) {
		await assert.rejects(
			resolveKennelAppPath({
				fs,
				appStatePath: "/Users/secret/.ao/app-state.json",
				platform: "darwin",
			}),
			(error) => {
				assert.ok(error instanceof KennelLaunchError);
				assert.equal(error.code, "KENNEL_APP_UNAVAILABLE");
				assert.doesNotMatch(error.message, /Users|secret|Applications/);
				return true;
			},
		);
	}

	await t.test("missing marker", () => expectUnavailable({
		readFile: async () => { throw Object.assign(new Error("/Users/secret missing"), { code: "ENOENT" }); },
		realpath: async () => assert.fail("realpath must not run"),
		stat: async () => assert.fail("stat must not run"),
	}));

	await t.test("renderer-style URL", () => expectUnavailable({
		readFile: async () => JSON.stringify({ appPath: "file:///Applications/Kennel.app" }),
		realpath: async () => assert.fail("realpath must not run"),
		stat: async () => assert.fail("stat must not run"),
	}));

	await t.test("non-application path", () => expectUnavailable({
		readFile: async () => JSON.stringify({ appPath: "/private/tmp/not-kennel" }),
		realpath: async () => assert.fail("realpath must not run"),
		stat: async () => assert.fail("stat must not run"),
	}));

	await t.test("symlink no longer resolves to an app", () => expectUnavailable({
		readFile: async () => JSON.stringify({ appPath: "/Applications/Kennel.app" }),
		realpath: async () => "/private/tmp/not-an-app",
		stat: async () => ({ isDirectory: () => true, isFile: () => false }),
	}));
});

test("non-macOS markers must resolve to a file rather than a directory", async () => {
	const fs = {
		readFile: async () => JSON.stringify({ appPath: "/opt/kennel/kennel" }),
		realpath: async () => "/opt/kennel/kennel",
		stat: async () => ({ isDirectory: () => false, isFile: () => true }),
	};
	assert.equal(
		await resolveKennelAppPath({
			fs,
			appStatePath: "/Users/tester/.ao/app-state.json",
			platform: "linux",
		}),
		"/opt/kennel/kennel",
	);
});
