import assert from "node:assert/strict";
import { EventEmitter } from "node:events";
import test from "node:test";
import {
	createHaptics,
	hapticsHelperPath,
	HAPTIC_THROTTLE_MS,
	isHapticPattern,
} from "./haptics.mjs";

function fakeHelper() {
	const written = [];
	const helper = new EventEmitter();
	helper.stdin = Object.assign(new EventEmitter(), {
		writable: true,
		write: (line) => written.push(line),
		end: () => {},
	});
	helper.kill = () => helper.emit("exit", 0);
	helper.written = written;
	return helper;
}

function harness({ helperExists = true, spawnThrows = false } = {}) {
	const spawned = [];
	let clock = 0;
	const helpers = [];

	const haptics = createHaptics({
		helperPath: "/helper",
		now: () => clock,
		fs: {
			access: async () => {
				if (!helperExists) throw new Error("ENOENT");
			},
		},
		spawn: (command) => {
			if (spawnThrows) throw new Error("spawn failed");
			spawned.push(command);
			const helper = fakeHelper();
			helpers.push(helper);
			return helper;
		},
	});

	return {
		haptics,
		spawned,
		helpers,
		advance: (ms) => {
			clock += ms;
		},
	};
}

test("the helper path lives beside the rest of the desktop code", () => {
	assert.equal(hapticsHelperPath("/Apps/Island.app"), "/Apps/Island.app/desktop/helpers/kennel-haptics");
});

test("only the three known patterns are patterns", () => {
	assert.equal(isHapticPattern("alignment"), true);
	assert.equal(isHapticPattern("level"), true);
	assert.equal(isHapticPattern("rumble"), false);
});

test("a tap writes one line to a helper spawned once", async () => {
	const { haptics, spawned, helpers, advance } = harness();

	assert.deepEqual(await haptics.perform("alignment"), { performed: true });
	advance(HAPTIC_THROTTLE_MS);
	assert.deepEqual(await haptics.perform("generic"), { performed: true });

	assert.equal(spawned.length, 1);
	assert.deepEqual(helpers[0].written, ["alignment\n", "generic\n"]);
});

test("an unknown pattern is sent as the softest one rather than refused", async () => {
	const { haptics, helpers } = harness();

	await haptics.perform("rumble");
	assert.deepEqual(helpers[0].written, ["alignment\n"]);
});

test("taps closer together than the throttle are dropped", async () => {
	const { haptics, helpers, advance } = harness();

	await haptics.perform();
	await haptics.perform();
	advance(HAPTIC_THROTTLE_MS - 1);
	await haptics.perform();
	advance(1);
	await haptics.perform();

	assert.equal(helpers[0].written.length, 2);
});

test("a missing helper is silent, and is not looked for twice", async () => {
	let checks = 0;
	const haptics = createHaptics({
		helperPath: "/helper",
		now: () => checks * 1000,
		fs: {
			access: async () => {
				checks += 1;
				throw new Error("ENOENT");
			},
		},
		spawn: () => assert.fail("must not spawn a helper that is not there"),
	});

	assert.deepEqual(await haptics.perform(), { performed: false });
	assert.deepEqual(await haptics.perform(), { performed: false });
	assert.equal(checks, 1);
});

test("a helper that fails to spawn is silent", async () => {
	const { haptics } = harness({ spawnThrows: true });
	assert.deepEqual(await haptics.perform(), { performed: false });
});

test("a helper that dies is replaced on the next tap", async () => {
	const { haptics, spawned, helpers, advance } = harness();

	await haptics.perform();
	helpers[0].emit("exit", 1);

	advance(HAPTIC_THROTTLE_MS);
	assert.deepEqual(await haptics.perform(), { performed: true });
	assert.equal(spawned.length, 2);
});

test("stopping ends the helper and leaves nothing behind to write to", async () => {
	const { haptics, helpers, advance } = harness();

	await haptics.perform();
	haptics.stop();
	advance(HAPTIC_THROTTLE_MS);
	await haptics.perform();

	// The first helper heard one tap; the replacement heard the second.
	assert.deepEqual(helpers[0].written, ["alignment\n"]);
	assert.deepEqual(helpers[1].written, ["alignment\n"]);
});
