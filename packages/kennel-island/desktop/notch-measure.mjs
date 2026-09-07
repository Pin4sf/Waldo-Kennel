// The measured notch, read from AppKit through the `kennel-notch` helper.
//
// `desktop/notch-geometry.mjs` derives the housing from the menu bar's height,
// which is accurate to a few points and was the best available without a native
// addon. AppKit knows the exact figure — `NSScreen.auxiliaryTopLeftArea` and
// its right-hand twin are the strips either side of the housing — so when the
// helper is present the island uses the measurement and the derivation becomes
// the fallback rather than the answer.
//
// Everything here is best-effort. A missing helper, a helper that fails, a
// helper that prints something unexpected: all of them mean "no measurement",
// and the caller keeps deriving.

import path from "node:path";

/** How long to wait for a helper that should answer in single-digit milliseconds. */
export const MEASURE_TIMEOUT_MS = 2_000;

export function notchHelperPath(appPath) {
	return path.join(appPath, "desktop", "helpers", "kennel-notch");
}

function positivePoints(value) {
	return Number.isFinite(value) && value > 0 && value < 2000 ? Math.round(value) : null;
}

/**
 * A helper's stdout, read as a measurement, or `null` when it is not one.
 *
 * A notchless display is a real answer — `{ hasNotch: false }` — and is
 * reported as such rather than as a failure, so a Mac with a flat bezel stops
 * the island guessing at a housing that is not there.
 */
export function parseMeasurement(stdout) {
	let parsed;
	try {
		parsed = JSON.parse(String(stdout));
	} catch {
		return null;
	}
	if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return null;
	if (parsed.hasNotch !== true) {
		return parsed.hasNotch === false ? { hasNotch: false } : null;
	}

	const notchWidth = positivePoints(parsed.notchWidth);
	const notchHeight = positivePoints(parsed.notchHeight);
	// A housing needs both dimensions. One of them alone is a partial answer,
	// and a partial answer is worse than the derivation it would replace.
	if (notchWidth === null || notchHeight === null) return null;

	return { hasNotch: true, notchWidth, notchHeight };
}

/**
 * Measures the built-in display once.
 *
 * Resolves to `null` whenever the measurement cannot be trusted, which is the
 * signal to fall back. It never rejects: the caller is startup.
 */
export function measureNotch({ execFile, helperPath, timeoutMs = MEASURE_TIMEOUT_MS }) {
	return new Promise((resolve) => {
		let settled = false;
		const finish = (value) => {
			if (settled) return;
			settled = true;
			resolve(value);
		};

		try {
			execFile(
				helperPath,
				[],
				{ timeout: timeoutMs, maxBuffer: 64 * 1024, windowsHide: true },
				(error, stdout) => finish(error ? null : parseMeasurement(stdout)),
			);
		} catch {
			finish(null);
		}
	});
}
