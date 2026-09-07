import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("the Island package is a renderer lab, not a second Electron application", async () => {
	const manifest = JSON.parse(await readFile(new URL("../package.json", import.meta.url), "utf8"));

	assert.equal(manifest.main, undefined);
	assert.equal(manifest.build, undefined);
	assert.equal(manifest.scripts.dev, "vite");
	for (const retired of ["dev:desktop", "start:desktop", "package:mac"]) {
		assert.equal(manifest.scripts[retired], undefined, `${retired} must not launch a second app`);
	}
	for (const retired of ["electron", "electron-builder", "concurrently", "wait-on"]) {
		assert.equal(manifest.devDependencies[retired], undefined, `${retired} belongs to the unified desktop host`);
	}
});

test("the unified Forge app builds and ships both Island helpers", async () => {
	const forgeConfig = await readFile(new URL("../../../frontend/forge.config.ts", import.meta.url), "utf8");
	const script = await readFile(new URL("../scripts/build-helpers.mjs", import.meta.url), "utf8");
	const produced = [...script.matchAll(/output:\s*"([^"]+)"/g)].map((match) => match[1]);

	assert.ok(produced.length > 0, "no helper outputs found in scripts/build-helpers.mjs");
	for (const name of produced) {
		assert.match(forgeConfig, new RegExp(`desktop/helpers/${name}`));
	}
	assert.match(forgeConfig, /entry:\s*"src\/island-preload\.ts"/);
	assert.match(forgeConfig, /name:\s*"island_window"/);
});

test("shared desktop sources target the current Kennel runtime identity", async () => {
	const runtimeHelpers = await readFile(new URL("./runtime-helpers.mjs", import.meta.url), "utf8");
	const kennelService = await readFile(new URL("./kennel-service.mjs", import.meta.url), "utf8");

	assert.match(runtimeHelpers, /env\.KENNEL_RUN_FILE/);
	assert.match(runtimeHelpers, /"\.kennel"/);
	assert.doesNotMatch(runtimeHelpers, /AO_RUN_FILE|"\.ao"/);
	assert.match(kennelService, /DAEMON_SERVICE_NAME = "kennel-daemon"/);
	assert.match(kennelService, /path\.join\(home, "\.kennel", "running\.json"\)/);
	assert.doesNotMatch(kennelService, /agent-orchestrator-daemon|path\.join\(home, "\.ao"/);
});
