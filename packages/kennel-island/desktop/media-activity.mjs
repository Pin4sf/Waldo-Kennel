// Media activity for the island's resting waveform and its hover ticker.
//
// Two separate questions, answered by two separate mechanisms, because macOS
// no longer answers them together:
//
//   Is audio playing?  Power assertions name every process holding the output
//                      device awake. Public, permission-free, and true for
//                      Music, Spotify, and a video in any browser alike.
//
//   What is playing?   `MRMediaRemoteGetNowPlayingInfo` — the API that would
//                      answer this for every source — has required the private
//                      `com.apple.mediaremote.now-playing-info` entitlement
//                      since macOS 15.4, and Apple does not grant it to third
//                      parties. Music and Spotify still answer over Apple
//                      events, so we ask them and nobody else.
//
// A browser therefore shows the waveform and a generic label. That is the
// honest floor: the island never invents a title it was not told.

const ASSERTIONS_ARGV = Object.freeze(["-g", "assertions"]);
const MAX_OUTPUT_BYTES = 512 * 1024;
const COMMAND_TIMEOUT_MS = 4_000;

// A call and a song are the same thing to CoreAudio: both run the output
// device. What separates them is the microphone. A conversation always opens
// audio-in alongside audio-out, so an assertion naming both directions is a
// call and never counts as media.
const RESOURCES_LINE = /Resources:\s*(.+)$/i;

// WebCore takes this only while an element is actually playing. The broader
// "WebKit Media Playback" assertion survives a pause, so it is not used: it
// would leave the waveform lit on any tab that merely has a video loaded.
const BROWSER_PLAYBACK_NAME = /com\.apple\.WebCore:\s*HTMLMediaElement playback/i;

const TRACK_SOURCES = Object.freeze([
	// `application "X" is running` is the one form that does not launch the app
	// it names, so a stopped player is never woken up just to be asked.
	{
		id: "music",
		application: "Music",
		script: 'if application "Music" is running then tell application "Music" to if player state is playing then return name of current track & "\\n" & artist of current track',
	},
	{
		id: "spotify",
		application: "Spotify",
		script: 'if application "Spotify" is running then tell application "Spotify" to if player state is playing then return name of current track & "\\n" & artist of current track',
	},
]);

/**
 * Whether `pmset -g assertions` output shows audio leaving the machine.
 *
 * A resource line belongs to the assertion named just above it, so the two are
 * read together rather than scanning for `audio-out` anywhere in the blob.
 */
export function parseAudioOutputActivity(output) {
	if (typeof output !== "string" || output.length === 0) return false;

	for (const line of output.split("\n")) {
		if (BROWSER_PLAYBACK_NAME.test(line)) return true;

		const resources = RESOURCES_LINE.exec(line);
		if (!resources) continue;

		const directions = resources[1].split(/\s+/);
		if (directions.includes("audio-out") && !directions.includes("audio-in")) return true;
	}
	return false;
}

/** Title and artist from a two-line Apple event reply, or null. */
export function parseTrackReply(raw) {
	if (typeof raw !== "string") return null;

	const [title, artist] = raw.split("\n").map((value) => value.trim());
	if (!title) return null;

	return {
		title: title.slice(0, 200),
		artist: artist ? artist.slice(0, 200) : "",
	};
}

function runCommand(execFile, command, argv) {
	return new Promise((resolve) => {
		execFile(
			command,
			argv,
			{ timeout: COMMAND_TIMEOUT_MS, maxBuffer: MAX_OUTPUT_BYTES, windowsHide: true },
			(error, stdout) => resolve(error ? null : String(stdout)),
		);
	});
}

/**
 * Current media state.
 *
 * `playing` is authoritative. `track` is best effort and stays null whenever
 * the source will not identify itself, which is every browser and every app
 * that has not been granted Automation access.
 */
export async function readMediaActivity({
	execFile,
	platform = process.platform,
	runningApplications = null,
} = {}) {
	if (platform !== "darwin" || typeof execFile !== "function") {
		return { playing: false, track: null };
	}

	const assertions = await runCommand(execFile, "pmset", ASSERTIONS_ARGV);
	const playing = parseAudioOutputActivity(assertions);
	if (!playing) return { playing: false, track: null };

	for (const source of TRACK_SOURCES) {
		// Asking a stopped app would launch it, so only running apps are asked.
		if (runningApplications && !runningApplications.includes(source.application)) continue;

		const reply = await runCommand(execFile, "osascript", ["-e", source.script]);
		const track = parseTrackReply(reply);
		if (track) return { playing: true, track: { ...track, source: source.id } };
	}

	return { playing: true, track: null };
}

/** Applications the Apple event probe is allowed to talk to. */
export function trackSourceApplications() {
	return TRACK_SOURCES.map((source) => source.application);
}

export { COMMAND_TIMEOUT_MS, TRACK_SOURCES };

/* --------------------------------------------------------------------------
   Transport
   --------------------------------------------------------------------------
   The horizontal swipe steps a track, and stepping a track needs a player that
   will take the instruction. That is the same short list that will name a
   track — Music and Spotify — and for the same reason: the general answer,
   `MRMediaRemoteSendCommand`, sits behind an entitlement Apple does not grant.

   A swipe over a browser therefore does nothing. The alternative is sending a
   system media key and hoping it lands on the right app, which on a machine
   with two players open is a coin toss, and a coin toss is worse than a
   gesture that reports honestly that it had no target.
   -------------------------------------------------------------------------- */

const TRANSPORT_COMMANDS = Object.freeze({
	next: "next track",
	previous: "previous track",
	"play-pause": "playpause",
});

export function isMediaCommand(value) {
	return typeof value === "string" && Object.hasOwn(TRANSPORT_COMMANDS, value);
}

/**
 * Sends a transport command to whichever known player is currently playing.
 *
 * Returns whether anything took it, so a caller can tell "no player" apart
 * from "the player refused", rather than reporting success into the void.
 */
export async function sendMediaCommand(command, {
	execFile,
	platform = process.platform,
	runningApplications = null,
} = {}) {
	if (platform !== "darwin" || typeof execFile !== "function") return { sent: false };
	if (!isMediaCommand(command)) return { sent: false };

	const verb = TRANSPORT_COMMANDS[command];

	for (const source of TRACK_SOURCES) {
		if (runningApplications && !runningApplications.includes(source.application)) continue;

		// Guarded on `is running` for the same reason the probe is: naming a
		// stopped app in an Apple event launches it, and a swipe must never
		// open a music player that nobody asked for.
		const script = `if application "${source.application}" is running then tell application "${source.application}" to if player state is playing then ${verb}`;
		const reply = await runCommand(execFile, "osascript", ["-e", script]);
		if (reply !== null) return { sent: true, source: source.id };
	}

	return { sent: false };
}
