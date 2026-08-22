import assert from "node:assert/strict";
import test from "node:test";
import * as nodeFs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import {
	createSettingsStore,
	defaultSettings,
	mergeSettings,
	normalizeSettings,
	sameSettings,
	SETTINGS_VERSION,
} from "./settings.mjs";

async function temporaryUserData() {
	return nodeFs.mkdtemp(path.join(os.tmpdir(), "kennel-island-settings-"));
}

test("defaults describe every section the schema declares", () => {
	const settings = defaultSettings();

	assert.equal(settings.version, SETTINGS_VERSION);
	assert.equal(settings.notch.widthOffset, 0);
	assert.equal(settings.notch.heightOffset, 0);
	assert.equal(settings.hover.peek, true);
	assert.equal(settings.hover.haptics, true);
	assert.equal(settings.gestures.enabled, true);
	assert.equal(settings.appearance.calibrating, false);
});

test("stored values outside their range are clamped rather than refused", () => {
	const settings = normalizeSettings({
		notch: { widthOffset: 900, heightOffset: -900, contentPadding: 7.6 },
	});

	assert.equal(settings.notch.widthOffset, 40);
	assert.equal(settings.notch.heightOffset, -12);
	assert.equal(settings.notch.contentPadding, 8);
});

test("values of the wrong type fall back to their default", () => {
	const settings = normalizeSettings({
		notch: { widthOffset: "not-a-number" },
		hover: { peek: "yes", haptics: null },
		gestures: "not-a-section",
	});

	assert.equal(settings.notch.widthOffset, 0);
	assert.equal(settings.hover.peek, true);
	assert.equal(settings.hover.haptics, true);
	assert.equal(settings.gestures.enabled, true);
});

test("numeric strings are accepted, because a range input reports one", () => {
	assert.equal(normalizeSettings({ notch: { widthOffset: "-6" } }).notch.widthOffset, -6);
});

test("unknown sections and unknown fields are dropped", () => {
	const settings = normalizeSettings({
		notch: { widthOffset: 4, somethingElse: true },
		invented: { field: 1 },
	});

	assert.equal(settings.notch.widthOffset, 4);
	assert.equal(Object.hasOwn(settings.notch, "somethingElse"), false);
	assert.equal(Object.hasOwn(settings, "invented"), false);
});

test("a patch changes only the fields it names", () => {
	const current = mergeSettings(defaultSettings(), { notch: { widthOffset: 8 } });
	const next = mergeSettings(current, { hover: { haptics: false } });

	assert.equal(next.notch.widthOffset, 8);
	assert.equal(next.hover.haptics, false);
	assert.equal(next.hover.peek, true);
});

test("sameSettings compares preferences, not object identity", () => {
	assert.equal(sameSettings(defaultSettings(), defaultSettings()), true);
	assert.equal(
		sameSettings(defaultSettings(), mergeSettings(defaultSettings(), { notch: { heightOffset: 2 } })),
		false,
	);
});

test("a store round-trips settings through the file it owns", async (t) => {
	const userDataPath = await temporaryUserData();
	t.after(() => nodeFs.rm(userDataPath, { recursive: true, force: true }));

	const store = createSettingsStore({ fs: nodeFs, userDataPath });
	await store.load();
	await store.update({ notch: { widthOffset: 6, heightOffset: 2 } });
	await store.flush();

	const reopened = createSettingsStore({ fs: nodeFs, userDataPath });
	const settings = await reopened.load();

	assert.equal(settings.notch.widthOffset, 6);
	assert.equal(settings.notch.heightOffset, 2);
});

test("a corrupt settings file loads as defaults instead of throwing", async (t) => {
	const userDataPath = await temporaryUserData();
	t.after(() => nodeFs.rm(userDataPath, { recursive: true, force: true }));

	const store = createSettingsStore({ fs: nodeFs, userDataPath });
	await nodeFs.writeFile(store.filePath, "{ not json", "utf8");

	assert.deepEqual(await store.load(), defaultSettings());
});

test("subscribers hear a change once, and not when nothing changed", async (t) => {
	const userDataPath = await temporaryUserData();
	t.after(() => nodeFs.rm(userDataPath, { recursive: true, force: true }));

	const store = createSettingsStore({ fs: nodeFs, userDataPath });
	const seen = [];
	store.onChange((settings) => seen.push(settings.notch.widthOffset));

	await store.load();
	await store.update({ notch: { widthOffset: 5 } });
	await store.update({ notch: { widthOffset: 5 } });
	await store.update({ notch: { widthOffset: 6 } });
	await store.flush();

	assert.deepEqual(seen, [5, 6]);
});

test("a burst of patches lands as one consistent file", async (t) => {
	const userDataPath = await temporaryUserData();
	t.after(() => nodeFs.rm(userDataPath, { recursive: true, force: true }));

	const store = createSettingsStore({ fs: nodeFs, userDataPath });
	await store.load();
	// A dragged slider produces exactly this: many patches, no awaiting.
	for (let offset = -10; offset <= 10; offset += 1) {
		void store.update({ notch: { widthOffset: offset } });
	}
	await store.flush();

	const reopened = createSettingsStore({ fs: nodeFs, userDataPath });
	assert.equal((await reopened.load()).notch.widthOffset, 10);
});
