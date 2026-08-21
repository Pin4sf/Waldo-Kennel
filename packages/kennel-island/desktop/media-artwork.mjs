// Album art for the island.
//
// The art is what the peek's colour is taken from, so it has to arrive as
// pixels the renderer can actually read. It arrives as a `data:` URI for two
// reasons: the renderer's CSP allows `img-src 'self' data:` and nothing else,
// and a canvas reading a data URI is never tainted, which is what lets the
// accent be sampled from it.
//
// The two players answer differently and neither answer is free:
//
//   Spotify   hands over an `artwork url` on its own CDN. Turning that into
//             pixels means one HTTPS GET, which is the only request this app
//             makes to anything. It is allowlisted by host, capped by size, and
//             switched off by a preference — see `media.artwork`.
//
//   Music     hands over the bytes directly, but only through an Apple event
//             that returns raw data, which `osascript` cannot print. The script
//             writes them to a file we name and we read the file back.
//
// Every failure resolves to `null`. Missing art is a wash we do not get, never
// an error the user has to see.

import path from "node:path";

/** Nothing larger than this is worth a wash behind two lines of text. */
export const MAX_ARTWORK_BYTES = 2 * 1024 * 1024;

export const ARTWORK_TIMEOUT_MS = 4_000;

/** Hosts the artwork fetch may talk to. Spotify's CDN, and nothing else. */
const ALLOWED_ARTWORK_HOSTS = Object.freeze(["i.scdn.co"]);

const ALLOWED_IMAGE_TYPES = Object.freeze([
	"image/jpeg",
	"image/png",
	"image/webp",
]);

/**
 * Whether a URL may be fetched for artwork.
 *
 * An allowlist rather than a scheme check: the URL arrives from another
 * application over an Apple event, and "some app told us to GET this" is not a
 * reason to GET anything.
 */
export function artworkUrlIsAllowed(rawUrl) {
	let url;
	try {
		url = new URL(String(rawUrl));
	} catch {
		return false;
	}

	return url.protocol === "https:" && ALLOWED_ARTWORK_HOSTS.includes(url.hostname);
}

export function isAllowedImageType(contentType) {
	if (typeof contentType !== "string") return false;
	return ALLOWED_IMAGE_TYPES.includes(contentType.split(";")[0].trim().toLowerCase());
}

export function toDataUri(contentType, bytes) {
	const type = String(contentType).split(";")[0].trim().toLowerCase();
	return `data:${type};base64,${Buffer.from(bytes).toString("base64")}`;
}

/**
 * Downloads artwork and returns it as a data URI, or null.
 *
 * The body is read as an ArrayBuffer and checked against the cap afterwards as
 * well as against the declared length, because a `content-length` is a claim
 * rather than a promise.
 */
export async function fetchArtwork(rawUrl, { fetchImpl, timeoutMs = ARTWORK_TIMEOUT_MS } = {}) {
	if (!artworkUrlIsAllowed(rawUrl) || typeof fetchImpl !== "function") return null;

	const controller = new AbortController();
	const timer = setTimeout(() => controller.abort(), timeoutMs);
	timer.unref?.();

	try {
		const response = await fetchImpl(String(rawUrl), {
			signal: controller.signal,
			redirect: "error",
		});
		if (!response?.ok) return null;

		const contentType = response.headers?.get?.("content-type");
		if (!isAllowedImageType(contentType)) return null;

		const declared = Number(response.headers?.get?.("content-length"));
		if (Number.isFinite(declared) && declared > MAX_ARTWORK_BYTES) return null;

		const bytes = new Uint8Array(await response.arrayBuffer());
		if (bytes.byteLength === 0 || bytes.byteLength > MAX_ARTWORK_BYTES) return null;

		return toDataUri(contentType, bytes);
	} catch {
		return null;
	} finally {
		clearTimeout(timer);
	}
}

/** Where the Music export lands. Named by us, so the script cannot choose it. */
export function musicArtworkPath(temporaryDirectory) {
	return path.join(temporaryDirectory, "kennel-island-artwork.data");
}

/**
 * Exports the current Music track's artwork and reads it back.
 *
 * The format has to be asked for separately: the raw data carries no type, and
 * a data URI with the wrong one will not decode.
 */
export async function readMusicArtwork({ execFile, fs, temporaryDirectory }) {
	if (typeof execFile !== "function") return null;

	const target = musicArtworkPath(temporaryDirectory);
	const script = `
if application "Music" is not running then return ""
tell application "Music"
	if player state is not playing then return ""
	if (count of artworks of current track) is 0 then return ""
	set theArtwork to artwork 1 of current track
	set theFormat to (format of theArtwork) as text
	set theData to raw data of theArtwork
end tell
set theFile to open for access POSIX file ${JSON.stringify(target)} with write permission
set eof theFile to 0
write theData to theFile
close access theFile
return theFormat
`.trim();

	const format = await new Promise((resolve) => {
		execFile(
			"osascript",
			["-e", script],
			{ timeout: ARTWORK_TIMEOUT_MS, maxBuffer: 64 * 1024, windowsHide: true },
			(error, stdout) => resolve(error ? null : String(stdout).trim()),
		);
	});
	if (!format) return null;

	try {
		const bytes = await fs.readFile(target);
		if (bytes.byteLength === 0 || bytes.byteLength > MAX_ARTWORK_BYTES) return null;
		return toDataUri(/png/i.test(format) ? "image/png" : "image/jpeg", bytes);
	} catch {
		return null;
	} finally {
		// The export is a scratch file with somebody's album art in it. It has
		// served its purpose the moment it is read.
		await fs.rm(target, { force: true }).catch(() => {});
	}
}

/**
 * The `artwork url` Spotify reports for what it is playing, or null.
 *
 * Guarded on `is running` for the same reason every other probe is: naming a
 * stopped app in an Apple event launches it.
 */
export function readSpotifyArtworkUrl({ execFile }) {
	if (typeof execFile !== "function") return Promise.resolve(null);

	const script =
		'if application "Spotify" is running then tell application "Spotify" to '
		+ "if player state is playing then return artwork url of current track";

	return new Promise((resolve) => {
		execFile(
			"osascript",
			["-e", script],
			{ timeout: ARTWORK_TIMEOUT_MS, maxBuffer: 8 * 1024, windowsHide: true },
			(error, stdout) => {
				const url = error ? "" : String(stdout).trim();
				resolve(artworkUrlIsAllowed(url) ? url : null);
			},
		);
	});
}

/**
 * Artwork for whichever player named the track, or null.
 *
 * `allowNetwork` is the user's preference reaching this far down. Spotify's
 * path is the only one that leaves the machine, so it is the only one gated.
 */
export async function readArtwork({
	source,
	execFile,
	fs,
	fetchImpl,
	temporaryDirectory,
	allowNetwork = true,
}) {
	if (source === "music") {
		return readMusicArtwork({ execFile, fs, temporaryDirectory });
	}
	if (source === "spotify") {
		if (!allowNetwork) return null;
		const url = await readSpotifyArtworkUrl({ execFile });
		return url ? fetchArtwork(url, { fetchImpl }) : null;
	}
	return null;
}

/** Whether two tracks are the same track, for deciding to re-read artwork. */
export function sameTrack(left, right) {
	return (
		(left?.title ?? null) === (right?.title ?? null) &&
		(left?.artist ?? null) === (right?.artist ?? null) &&
		(left?.source ?? null) === (right?.source ?? null)
	);
}
