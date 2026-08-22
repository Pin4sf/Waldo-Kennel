// User settings for the island.
//
// Two rules shape this file.
//
// First, the schema is data, not code: every field is described once, in
// `SETTINGS_SCHEMA`, and reading, clamping, merging and persisting are all
// driven off that description. Adding a preference is adding a row.
//
// Second, a settings file is untrusted input. It is written by this app, but it
// lives in a directory the user can edit and another process can corrupt, so
// every value that comes back off disk is validated against the schema and
// silently replaced by its default when it does not fit. A malformed file
// degrades to defaults; it never throws on a path that would stop the island
// from starting.

import path from "node:path";

/** Bump when a stored shape can no longer be migrated field-by-field. */
export const SETTINGS_VERSION = 1;

const boolean = (fallback) => ({ kind: "boolean", fallback });
const integer = (fallback, min, max) => ({ kind: "integer", fallback, min, max });
const choice = (fallback, values) => ({ kind: "choice", fallback, values: Object.freeze(values) });

/**
 * Every preference the island understands.
 *
 * Ranges are points, and they are deliberately narrow. The notch fine tune
 * corrects a derived estimate that is already accurate to a few points, so a
 * slider that can travel 200pt would only offer new ways to be wrong.
 */
export const SETTINGS_SCHEMA = Object.freeze({
	notch: Object.freeze({
		/** Points added to each side of the derived notch width. */
		widthOffset: integer(0, -40, 40),
		/** Points added to the derived notch height. */
		heightOffset: integer(0, -12, 24),
		/** Horizontal breathing room between the housing and the clusters. */
		contentPadding: integer(12, 0, 32),
	}),
	hover: Object.freeze({
		/** The dormant notch swells slightly under the pointer before it opens. */
		peek: boolean(true),
		/** Points the peek adds to each side of the housing. */
		peekWidth: integer(14, 0, 48),
		/** Points the peek adds below the housing. */
		peekHeight: integer(6, 0, 24),
		/** Pointer dwell before the peek commits, in milliseconds. */
		peekDelayMs: integer(90, 0, 600),
		/** Skip the peek and open the full panel on hover alone. */
		openOnHover: boolean(false),
		/** Keep an open panel up after the pointer leaves. */
		holdOnMouseLeave: boolean(false),
		/** Force Touch feedback when the island changes state under the pointer. */
		haptics: boolean(true),
	}),
	gestures: Object.freeze({
		enabled: boolean(true),
		/** Two-finger vertical swipe opens and closes. */
		verticalOpenClose: boolean(true),
		/** Two-finger horizontal swipe steps the track. */
		horizontalMedia: boolean(true),
		/** Swap which horizontal direction means "next". */
		invertMedia: boolean(false),
	}),
	media: Object.freeze({
		/**
		 * Fetch album art from the player's own CDN.
		 *
		 * The only outbound request this app makes. Spotify names an artwork URL
		 * rather than handing over the bytes, so the picture — and the accent
		 * colour taken from it — costs one HTTPS GET per track. Music's artwork
		 * is local and is read whatever this is set to.
		 */
		artwork: boolean(true),
		/** Animate the waveform while something is playing. */
		waveform: boolean(true),
	}),
	appearance: Object.freeze({
		/** Render the notch calibration outline over the housing. */
		calibrating: boolean(false),
		/** Keep the island awake so a screen recording can catch it. */
		demoMode: boolean(false),
	}),
});

const SECTIONS = Object.freeze(Object.keys(SETTINGS_SCHEMA));

function clampInteger(value, field) {
	const parsed = typeof value === "string" ? Number.parseFloat(value) : value;
	if (!Number.isFinite(parsed)) return field.fallback;
	return Math.min(field.max, Math.max(field.min, Math.round(parsed)));
}

function coerce(value, field) {
	switch (field.kind) {
		case "boolean":
			return typeof value === "boolean" ? value : field.fallback;
		case "integer":
			return clampInteger(value, field);
		case "choice":
			return field.values.includes(value) ? value : field.fallback;
		default:
			return field.fallback;
	}
}

/** The settings an untouched install runs on. */
export function defaultSettings() {
	const settings = { version: SETTINGS_VERSION };
	for (const section of SECTIONS) {
		settings[section] = {};
		for (const [name, field] of Object.entries(SETTINGS_SCHEMA[section])) {
			settings[section][name] = field.fallback;
		}
	}
	return settings;
}

function plainObject(value) {
	return value && typeof value === "object" && !Array.isArray(value) ? value : {};
}

/**
 * Every value in `input` that the schema recognises, clamped to its range;
 * everything else replaced by its default. Unknown sections and unknown fields
 * are dropped rather than carried, so a settings file written by a newer build
 * cannot smuggle a key past validation.
 */
export function normalizeSettings(input) {
	const source = plainObject(input);
	const settings = { version: SETTINGS_VERSION };

	for (const section of SECTIONS) {
		const stored = plainObject(source[section]);
		settings[section] = {};
		for (const [name, field] of Object.entries(SETTINGS_SCHEMA[section])) {
			settings[section][name] = Object.hasOwn(stored, name)
				? coerce(stored[name], field)
				: field.fallback;
		}
	}

	return settings;
}

/**
 * `current` with `patch` applied. A patch names only what it changes: absent
 * sections and absent fields keep the value they already had, which is what
 * lets a settings pane send one slider rather than the whole document.
 */
export function mergeSettings(current, patch) {
	const base = normalizeSettings(current);
	const changes = plainObject(patch);
	const merged = { version: SETTINGS_VERSION };

	for (const section of SECTIONS) {
		const incoming = plainObject(changes[section]);
		merged[section] = { ...base[section] };
		for (const [name, field] of Object.entries(SETTINGS_SCHEMA[section])) {
			if (!Object.hasOwn(incoming, name)) continue;
			merged[section][name] = coerce(incoming[name], field);
		}
	}

	return merged;
}

/** Whether two normalized settings documents describe the same preferences. */
export function sameSettings(left, right) {
	for (const section of SECTIONS) {
		for (const name of Object.keys(SETTINGS_SCHEMA[section])) {
			if (left?.[section]?.[name] !== right?.[section]?.[name]) return false;
		}
	}
	return true;
}

export function settingsFilePath(userDataPath) {
	return path.join(userDataPath, "settings.json");
}

/**
 * A settings store backed by one JSON file.
 *
 * Every read and every write runs on one promise chain. A slider dragged across
 * its range produces a burst of patches with nothing awaiting them, and two
 * overlapping writers would race for the same path — so mutations queue rather
 * than overlap, and `flush()` is the tail of that queue.
 *
 * Writes go to a sibling temp file and are renamed into place, so a crash
 * mid-write leaves the previous settings intact rather than a truncated file
 * the next launch would have to discard.
 */
export function createSettingsStore({ fs, userDataPath }) {
	const filePath = settingsFilePath(userDataPath);
	const listeners = new Set();
	let cached = defaultSettings();
	let loaded = false;
	let queue = Promise.resolve();

	function enqueue(work) {
		// Both arms run `work`: one failed mutation must not strand every later
		// one behind a rejected link in the chain.
		const running = queue.then(work, work);
		queue = running.then(
			() => undefined,
			() => undefined,
		);
		return running;
	}

	async function readFromDisk() {
		if (loaded) return cached;
		loaded = true;

		try {
			const raw = await fs.readFile(filePath, "utf8");
			cached = normalizeSettings(JSON.parse(raw));
		} catch {
			// A missing file is the first launch, and an unreadable or malformed
			// one is a file we are about to overwrite anyway. Neither is worth
			// refusing to start over.
			cached = defaultSettings();
		}
		return cached;
	}

	async function persist(settings) {
		const temporaryPath = `${filePath}.${process.pid}.tmp`;
		try {
			await fs.mkdir(path.dirname(filePath), { recursive: true });
			await fs.writeFile(temporaryPath, `${JSON.stringify(settings, null, "\t")}\n`, {
				encoding: "utf8",
				mode: 0o600,
			});
			await fs.rename(temporaryPath, filePath);
		} catch {
			// Settings that cannot be written still apply for this session. The
			// alternative — surfacing a modal over the notch — is worse than
			// quietly running on the values the user just chose.
		}
	}

	return {
		filePath,
		/** The settings in memory. Valid before `load()` too: defaults until then. */
		current() {
			return cached;
		},
		load() {
			return enqueue(readFromDisk);
		},
		update(patch) {
			return enqueue(async () => {
				await readFromDisk();
				const next = mergeSettings(cached, patch);
				if (sameSettings(next, cached)) return cached;

				cached = next;
				await persist(next);
				for (const listener of listeners) listener(cached);
				return cached;
			});
		},
		reset() {
			return this.update(defaultSettings());
		},
		onChange(listener) {
			listeners.add(listener);
			return () => listeners.delete(listener);
		},
		/** Resolves once every queued read and write has landed. */
		flush() {
			return queue;
		},
	};
}
