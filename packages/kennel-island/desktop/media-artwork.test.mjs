import assert from "node:assert/strict";
import test from "node:test";
import {
	artworkUrlIsAllowed,
	fetchArtwork,
	isAllowedImageType,
	MAX_ARTWORK_BYTES,
	musicArtworkPath,
	readArtwork,
	sameTrack,
	toDataUri,
} from "./media-artwork.mjs";

function imageResponse({
	ok = true,
	contentType = "image/jpeg",
	bytes = new Uint8Array([1, 2, 3]),
	contentLength = null,
} = {}) {
	return {
		ok,
		headers: {
			get: (name) => {
				if (name === "content-type") return contentType;
				if (name === "content-length") return contentLength;
				return null;
			},
		},
		arrayBuffer: async () => bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength),
	};
}

test("only Spotify's own CDN over HTTPS may be fetched", () => {
	assert.equal(artworkUrlIsAllowed("https://i.scdn.co/image/abc"), true);
	assert.equal(artworkUrlIsAllowed("http://i.scdn.co/image/abc"), false);
	assert.equal(artworkUrlIsAllowed("https://evil.example/image/abc"), false);
	// A lookalike host is still not the host.
	assert.equal(artworkUrlIsAllowed("https://i.scdn.co.evil.example/x"), false);
	assert.equal(artworkUrlIsAllowed("file:///etc/passwd"), false);
	assert.equal(artworkUrlIsAllowed("not a url"), false);
	assert.equal(artworkUrlIsAllowed(undefined), false);
});

test("only real image types are accepted", () => {
	assert.equal(isAllowedImageType("image/jpeg"), true);
	assert.equal(isAllowedImageType("image/png; charset=binary"), true);
	assert.equal(isAllowedImageType("text/html"), false);
	assert.equal(isAllowedImageType("image/svg+xml"), false);
	assert.equal(isAllowedImageType(null), false);
});

test("a fetched image becomes a data URI", async () => {
	const artwork = await fetchArtwork("https://i.scdn.co/image/abc", {
		fetchImpl: async () => imageResponse(),
	});

	assert.equal(artwork, toDataUri("image/jpeg", new Uint8Array([1, 2, 3])));
});

test("a disallowed URL is never requested", async () => {
	const artwork = await fetchArtwork("https://evil.example/image", {
		fetchImpl: () => assert.fail("must not fetch a host outside the allowlist"),
	});

	assert.equal(artwork, null);
});

test("a response that is not an image is refused", async () => {
	assert.equal(
		await fetchArtwork("https://i.scdn.co/image/abc", {
			fetchImpl: async () => imageResponse({ contentType: "text/html" }),
		}),
		null,
	);
	assert.equal(
		await fetchArtwork("https://i.scdn.co/image/abc", {
			fetchImpl: async () => imageResponse({ ok: false }),
		}),
		null,
	);
});

test("an oversized image is refused by its claim and by its body", async () => {
	// The declared length is a claim.
	assert.equal(
		await fetchArtwork("https://i.scdn.co/image/abc", {
			fetchImpl: async () => imageResponse({ contentLength: String(MAX_ARTWORK_BYTES + 1) }),
		}),
		null,
	);
	// A body that lied about it is checked too.
	assert.equal(
		await fetchArtwork("https://i.scdn.co/image/abc", {
			fetchImpl: async () => imageResponse({
				contentLength: "10",
				bytes: new Uint8Array(MAX_ARTWORK_BYTES + 1),
			}),
		}),
		null,
	);
});

test("an empty body is not artwork", async () => {
	assert.equal(
		await fetchArtwork("https://i.scdn.co/image/abc", {
			fetchImpl: async () => imageResponse({ bytes: new Uint8Array(0) }),
		}),
		null,
	);
});

test("a fetch that throws resolves to no artwork rather than rejecting", async () => {
	assert.equal(
		await fetchArtwork("https://i.scdn.co/image/abc", {
			fetchImpl: async () => {
				throw new Error("offline");
			},
		}),
		null,
	);
});

test("the Music export lands on a path we chose, not one a script did", () => {
	assert.equal(musicArtworkPath("/tmp"), "/tmp/kennel-island-artwork.data");
});

test("Spotify's artwork is not fetched when the network is switched off", async () => {
	const artwork = await readArtwork({
		source: "spotify",
		allowNetwork: false,
		execFile: () => assert.fail("must not probe when artwork is off"),
		fetchImpl: () => assert.fail("must not fetch when artwork is off"),
	});

	assert.equal(artwork, null);
});

test("a source with no artwork path returns nothing", async () => {
	assert.equal(await readArtwork({ source: "browser", execFile: () => {} }), null);
	assert.equal(await readArtwork({ source: undefined, execFile: () => {} }), null);
});

test("Music's artwork is read locally, with no network involved", async () => {
	const removed = [];
	const artwork = await readArtwork({
		source: "music",
		temporaryDirectory: "/tmp",
		execFile: (_command, _args, _options, callback) => callback(null, "JPEG picture"),
		fetchImpl: () => assert.fail("Music's artwork must never leave the machine"),
		fs: {
			readFile: async () => Buffer.from([9, 8, 7]),
			rm: async (target) => {
				removed.push(target);
			},
		},
	});

	assert.equal(artwork, toDataUri("image/jpeg", new Uint8Array([9, 8, 7])));
	// The export is somebody's album art in a scratch file; it does not linger.
	assert.deepEqual(removed, ["/tmp/kennel-island-artwork.data"]);
});

test("a Music track with no artwork reports none", async () => {
	const artwork = await readArtwork({
		source: "music",
		temporaryDirectory: "/tmp",
		execFile: (_command, _args, _options, callback) => callback(null, ""),
		fs: { readFile: async () => assert.fail("nothing was exported"), rm: async () => {} },
	});

	assert.equal(artwork, null);
});

test("tracks are the same only when title, artist, and player all match", () => {
	const track = { title: "Challenge", artist: "Young Thug", source: "spotify" };

	assert.equal(sameTrack(track, { ...track }), true);
	assert.equal(sameTrack(track, { ...track, title: "Other" }), false);
	assert.equal(sameTrack(track, { ...track, artist: "Other" }), false);
	assert.equal(sameTrack(track, { ...track, source: "music" }), false);
	assert.equal(sameTrack(null, null), true);
	assert.equal(sameTrack(track, null), false);
});
