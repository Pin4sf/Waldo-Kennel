import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import vm from "node:vm";

const stageGeometryReply = {
	stageWidth: 760,
	stageHeight: 460,
	hasNotch: true,
	notchWidth: 200,
	notchHeight: 37,
	menuBarHeight: 37,
	scaleFactor: 2,
};

async function loadPreload() {
	const source = await readFile(new URL("./preload.cjs", import.meta.url), "utf8");
	const calls = [];
	const listeners = [];
	let exposed;

	const invokeResult = (channel) =>
		channel === "kennel-island:get-stage-geometry" ? stageGeometryReply : { ok: true };
	const electron = {
		contextBridge: {
			exposeInMainWorld(name, value) {
				exposed = { name, value };
			},
		},
		ipcRenderer: {
			invoke(...args) {
				calls.push(args);
				return Promise.resolve(invokeResult(args[0]));
			},
			on(channel, listener) {
				listeners.push({ channel, listener });
			},
			removeListener(channel, listener) {
				const index = listeners.findIndex(
					(entry) => entry.channel === channel && entry.listener === listener,
				);
				if (index >= 0) listeners.splice(index, 1);
			},
		},
	};
	const context = vm.createContext({
		require(specifier) {
			assert.equal(specifier, "electron");
			return electron;
		},
	});
	vm.runInContext(source, context, { filename: "desktop/preload.cjs" });
	return { api: exposed.value, exposedName: exposed.name, calls, listeners };
}

test("the preload conversation bridge sends only the typed session identifier", async () => {
	const { api, exposedName, calls } = await loadPreload();
	assert.equal(exposedName, "kennelDesktop");
	assert.equal(typeof api.getKennelConversation, "function");

	await api.getKennelConversation({
		sessionId: "worker-1",
		command: "must not cross preload",
		path: "/private/value",
	});

	assert.equal(JSON.stringify(calls), JSON.stringify([
		["kennel-island:get-conversation", { sessionId: "worker-1" }],
	]));
});

test("the preload exposes no generic IPC primitive", async () => {
	const { api } = await loadPreload();
	assert.equal("invoke" in api, false);
	assert.equal("send" in api, false);
	assert.equal("ipcRenderer" in api, false);
});

test("snapshot invalidation is a payload-free subscription that can be released", async () => {
	const { api, listeners } = await loadPreload();
	let calls = 0;
	const unsubscribe = api.onKennelSnapshotInvalidated(() => { calls += 1; });

	assert.equal(listeners.at(-1).channel, "kennel-island:snapshot-invalidated");
	listeners.at(-1).listener({ sender: "must not cross" }, { injected: true });
	assert.equal(calls, 1);
	unsubscribe();
	assert.equal(listeners.length, 0);
});

test("interactivity crosses the bridge as a boolean the renderer cannot widen", async () => {
	const { api, calls } = await loadPreload();

	await api.setInteractive("yes");
	await api.setInteractive(true);

	assert.equal(JSON.stringify(calls), JSON.stringify([
		["kennel-island:set-interactive", { interactive: false }],
		["kennel-island:set-interactive", { interactive: true }],
	]));
});

test("hiding the island uses one dedicated host action", async () => {
	const { api, calls } = await loadPreload();

	await api.hideIsland();

	assert.deepEqual(structuredClone(calls), [["kennel-island:hide-island"]]);
});

test("stage geometry arrives as finite measurements with a known shape", async () => {
	const { api } = await loadPreload();

	assert.deepEqual({ ...(await api.getStageGeometry()) }, stageGeometryReply);
});

test("a stage geometry subscription forwards only the payload and can be released", async () => {
	const { api, listeners } = await loadPreload();
	const received = [];

	const unsubscribe = api.onStageGeometry((geometry) => received.push(geometry));
	assert.equal(listeners.length, 1);
	assert.equal(listeners[0].channel, "kennel-island:stage-geometry-changed");

	listeners[0].listener({ sender: "must not cross" }, { ...stageGeometryReply, notchWidth: "wide" });
	assert.equal(received.length, 1);
	assert.equal(received[0].notchWidth, 0);
	assert.equal(received[0].notchHeight, 37);
	assert.equal("sender" in received[0], false);

	unsubscribe();
	assert.equal(listeners.length, 0);
});

test("a non-callable stage geometry subscription is inert", async () => {
	const { api, listeners } = await loadPreload();

	const unsubscribe = api.onStageGeometry(undefined);
	assert.equal(listeners.length, 0);
	assert.equal(typeof unsubscribe, "function");
	unsubscribe();
});

test("a track name crosses the bridge as bounded text and nothing else", async () => {
	const { api, listeners } = await loadPreload();
	const received = [];

	api.onMediaActivity((activity) => received.push(activity));
	listeners.at(-1).listener({}, {
		playing: true,
		track: { title: "x".repeat(500), artist: { toString: () => "injected" } },
		extra: "must not cross",
	});

	assert.equal(received[0].playing, true);
	assert.equal(received[0].track.title.length, 200);
	assert.equal(received[0].track.artist, "");
	assert.equal("extra" in received[0], false);
});

test("media with no identifiable track carries a null track, not a stub", async () => {
	const { api, listeners } = await loadPreload();
	const received = [];

	api.onMediaActivity((activity) => received.push(activity));
	listeners.at(-1).listener({}, { playing: true, track: null });

	assert.equal(received[0].playing, true);
	assert.equal(received[0].track, null);
});

test("settings cross the bridge as a known shape, with unknown keys dropped", async () => {
	const { api, listeners } = await loadPreload();
	const received = [];

	api.onSettings((settings) => received.push(settings));
	listeners.at(-1).listener({}, {
		notch: { widthOffset: 4, injected: "must not cross" },
		hover: { peek: "not a boolean" },
		invented: { field: 1 },
	});

	const settings = received[0];
	assert.equal(settings.notch.widthOffset, 4);
	assert.equal("injected" in settings.notch, false);
	assert.equal("invented" in settings, false);
	// A field of the wrong type is the default, never the wrong type.
	assert.equal(settings.hover.peek, true);
	assert.equal(settings.gestures.enabled, true);
});

test("a settings patch carries only the fields the caller actually named", async () => {
	const { api, calls } = await loadPreload();

	await api.updateSettings({
		notch: { widthOffset: 6, injected: true },
		hover: { haptics: false, peekWidth: "12" },
		invented: { field: 1 },
	});

	const [channel, payload] = calls.at(-1);
	assert.equal(channel, "kennel-island:update-settings");
	// The payload is built inside the preload's realm, so it is compared by
	// value rather than by prototype.
	assert.deepEqual(structuredClone(payload), { notch: { widthOffset: 6 }, hover: { haptics: false } });
});

test("the haptic bridge forwards only a pattern the island could have produced", async () => {
	const { api, calls } = await loadPreload();

	await api.performHaptic("level");
	assert.deepEqual(structuredClone(calls.at(-1)), ["kennel-island:perform-haptic", { pattern: "level" }]);

	await api.performHaptic("rumble");
	assert.deepEqual(structuredClone(calls.at(-1)), ["kennel-island:perform-haptic", { pattern: "alignment" }]);
});
