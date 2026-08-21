import assert from "node:assert/strict";
import { readdir, readFile, stat } from "node:fs/promises";
import test from "node:test";

// electron-builder ships an explicit allowlist rather than the whole tree, so a
// new main-process module is absent from the packaged app until it is named
// here. Nothing in development notices: `electron .` reads the working
// directory, and only the packaged build fails, at launch, on a missing import.
test("every main-process module is listed in the packaged file allowlist", async () => {
	const manifest = JSON.parse(await readFile(new URL("../package.json", import.meta.url), "utf8"));
	const shipped = new Set(manifest.build.files);

	const modules = (await readdir(new URL(".", import.meta.url)))
		.filter((name) => /\.(mjs|cjs)$/.test(name) && !name.endsWith(".test.mjs"));

	const missing = modules.filter((name) => !shipped.has(`desktop/${name}`));
	assert.deepEqual(missing, [], `not packaged: ${missing.join(", ")}`);
});

test("the allowlist names no main-process file that no longer exists", async () => {
	const manifest = JSON.parse(await readFile(new URL("../package.json", import.meta.url), "utf8"));

	const stale = [];
	for (const entry of manifest.build.files) {
		if (!entry.startsWith("desktop/")) continue;
		try {
			await stat(new URL(`../${entry}`, import.meta.url));
		} catch {
			stale.push(entry);
		}
	}
	assert.deepEqual(stale, [], `listed but absent: ${stale.join(", ")}`);
});

// The Swift helpers are optional to build — a machine with no toolchain skips
// them — but "optional to build" must not quietly become "never shipped". Every
// helper the build script produces has to be named in the allowlist, or the
// packaged app silently loses haptics and falls back to guessing the notch.
test("every compiled helper the build script produces is packaged", async () => {
	const manifest = JSON.parse(await readFile(new URL("../package.json", import.meta.url), "utf8"));
	const script = await readFile(new URL("../scripts/build-helpers.mjs", import.meta.url), "utf8");

	const produced = [...script.matchAll(/output:\s*"([^"]+)"/g)].map((match) => match[1]);
	assert.ok(produced.length > 0, "no helper outputs found in scripts/build-helpers.mjs");

	const missing = produced.filter((name) => !manifest.build.files.includes(`desktop/helpers/${name}`));
	assert.deepEqual(missing, [], `not packaged: ${missing.join(", ")}`);
});

test("test files are never packaged", async () => {
	const manifest = JSON.parse(await readFile(new URL("../package.json", import.meta.url), "utf8"));

	assert.equal(manifest.build.files.some((entry) => entry.includes(".test.")), false);
});
