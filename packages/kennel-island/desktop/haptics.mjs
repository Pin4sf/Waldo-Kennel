// Force Touch feedback, spoken to the helper in `desktop/helpers/`.
//
// Everything here is best-effort. Haptics only exist on a Force Touch trackpad,
// the helper only exists when a Swift toolchain was present at build time, and
// neither absence is worth a single line of user-visible error — so every
// failure path ends in "no tap happened", not an exception.
//
// The helper is spawned once and kept, because feedback that arrives late reads
// as an unrelated event rather than as the shape moving. It is also spawned
// lazily: a user who never hovers the island never starts a second process.

import path from "node:path";

/** Patterns the helper understands. Anything else is read as `alignment`. */
export const HAPTIC_PATTERNS = Object.freeze(["alignment", "generic", "level"]);

/** Minimum gap between taps. Below this the hand feels a buzz, not an event. */
export const HAPTIC_THROTTLE_MS = 120;

export function isHapticPattern(value) {
	return HAPTIC_PATTERNS.includes(value);
}

export function hapticsHelperPath(appPath) {
	return path.join(appPath, "desktop", "helpers", "kennel-haptics");
}

/**
 * A haptics channel.
 *
 * `perform` never throws and never reports failure: the caller is a hover
 * handler, and there is nothing it could usefully do with the news that a
 * trackpad did not buzz.
 */
export function createHaptics({ spawn, fs, helperPath, now = () => Date.now() }) {
	let helper = null;
	let unavailable = false;
	let lastTapAt = Number.NEGATIVE_INFINITY;

	function drop() {
		helper = null;
	}

	async function ensureHelper() {
		if (helper) return helper;
		if (unavailable) return null;

		try {
			await fs.access(helperPath);
		} catch {
			// Built without a Swift toolchain. Stop checking: the file will not
			// appear while the app is running.
			unavailable = true;
			return null;
		}

		try {
			helper = spawn(helperPath, [], { stdio: ["pipe", "ignore", "ignore"] });
		} catch {
			unavailable = true;
			return null;
		}

		helper.once("exit", drop);
		helper.once("error", drop);
		// A helper that dies mid-write must not take the main process with it.
		helper.stdin?.on("error", () => {});
		return helper;
	}

	return {
		async perform(pattern = "alignment") {
			const at = now();
			if (at - lastTapAt < HAPTIC_THROTTLE_MS) return { performed: false };
			lastTapAt = at;

			const process = await ensureHelper();
			if (!process?.stdin?.writable) return { performed: false };

			try {
				process.stdin.write(`${isHapticPattern(pattern) ? pattern : "alignment"}\n`);
			} catch {
				drop();
				return { performed: false };
			}
			return { performed: true };
		},
		stop() {
			if (!helper) return;
			const running = helper;
			drop();
			try {
				running.stdin?.end();
				running.kill();
			} catch {
				// Already gone.
			}
		},
	};
}
