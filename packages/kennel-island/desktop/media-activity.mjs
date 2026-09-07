// Media activity for the island's resting waveform and its hover ticker.
//
// Three separate questions, answered by three separate mechanisms, because
// macOS no longer answers them together:
//
//   Is audio playing?  Power assertions name every process holding the output
//                      device awake. Public, permission-free, and true for
//                      Music, Spotify, and a video in any browser alike.
//
//   Who is playing it? The same assertions name their owning process, so the
//                      island can say "Safari" even when it cannot say what
//                      Safari is playing — and can bring that app forward when
//                      the artwork is clicked.
//
//   What is playing?   `MRMediaRemoteGetNowPlayingInfo` — the API that would
//                      answer this for every source — has required the private
//                      `com.apple.mediaremote.now-playing-info` entitlement
//                      since macOS 15.4, and Apple does not grant it to third
//                      parties. So the owner is asked directly: desktop players
//                      over their own Apple event vocabulary, and browsers for
//                      the title of the tab making the noise, which is the only
//                      thing a browser will say about its own playback.
//
// A source that answers none of that still gets a waveform and an honest
// "Audio playing". The island never invents a title it was not told, and never
// keeps showing the last one it was.

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

/** `pid 76096(Safari):` — the process an assertion line belongs to. */
const OWNER_LINE = /^\s*pid\s+\d+\(([^)]+)\)/;

// coreaudiod owns the speaker assertion on behalf of whoever is playing, so it
// names the audio system rather than the source. Naming it would put "Core
// Audio" on the island; better to report no owner and let the probe find one.
const ANONYMOUS_OWNERS = new Set(["coreaudiod", "runningboardd", "audioaccessoryd"]);

/**
 * Desktop players that will name their own track.
 *
 * `application "X" is running` is the one form that does not launch the app it
 * names, so a stopped player is never woken up just to be asked.
 */
// `durationUnit` is not pedantry. Music reports `duration` in seconds and
// Spotify reports it in milliseconds, from an identically named property, so a
// single reader would put a four-minute song at four milliseconds on one of
// them and never notice until a progress bar was drawn against it.
const TRACK_SOURCES = Object.freeze([
	{
		id: "music",
		application: "Music",
		kind: "player",
		track: 'name of current track & "\\n" & artist of current track',
		position: "player position",
		duration: "duration of current track",
		durationUnit: "s",
		seekable: true,
	},
	{
		id: "spotify",
		application: "Spotify",
		kind: "player",
		track: 'name of current track & "\\n" & artist of current track',
		position: "player position",
		duration: "duration of current track",
		durationUnit: "ms",
		seekable: true,
	},
	{
		id: "tidal",
		application: "TIDAL",
		kind: "player",
		track: 'name of current track & "\\n" & artist of current track',
	},
	{
		id: "swinsian",
		application: "Swinsian",
		kind: "player",
		track: 'name of current track & "\\n" & artist of current track',
		position: "player position",
		duration: "duration of current track",
		durationUnit: "s",
		seekable: true,
	},
	{
		id: "doppler",
		application: "Doppler",
		kind: "player",
		track: 'name of current track & "\\n" & artist of current track',
	},
	// VLC and IINA play files, so there is a name and no artist. The second
	// line is deliberately empty rather than absent: `parseTrackReply` reads a
	// two-line reply, and a missing line would make the parse depend on which
	// player answered.
	{ id: "vlc", application: "VLC", kind: "player", track: 'name of current item & "\\n"', state: "playing" },
	{ id: "iina", application: "IINA", kind: "player", track: 'name of current item & "\\n"', state: "playing" },
]);

/**
 * Browsers, which will not say what is playing but will say what the tab is
 * called — and a tab making noise is almost always called after it.
 *
 * Firefox is absent on purpose: it ships no Apple event vocabulary at all, so
 * there is nothing to ask. It still gets the waveform and "Audio playing".
 */
const BROWSER_SOURCES = Object.freeze([
	{ id: "safari", application: "Safari", kind: "browser", track: "name of current tab of front window" },
	{ id: "chrome", application: "Google Chrome", kind: "browser", track: "title of active tab of front window" },
	{ id: "brave", application: "Brave Browser", kind: "browser", track: "title of active tab of front window" },
	{ id: "edge", application: "Microsoft Edge", kind: "browser", track: "title of active tab of front window" },
	{ id: "vivaldi", application: "Vivaldi", kind: "browser", track: "title of active tab of front window" },
	{ id: "chromium", application: "Chromium", kind: "browser", track: "title of active tab of front window" },
	{ id: "arc", application: "Arc", kind: "browser", track: "title of active tab of front window" },
]);

const ALL_SOURCES = Object.freeze([...TRACK_SOURCES, ...BROWSER_SOURCES]);

/** Trailing site furniture a browser tab drags along. */
const TAB_TITLE_SUFFIX = /\s*[-–|—]\s*(YouTube|SoundCloud|Spotify|Bandcamp|Apple Music|Vimeo|Twitch|Netflix|Mixcloud|Deezer)\s*$/i;
/** Unread counts a tab title carries while it plays. */
const TAB_TITLE_BADGE = /^\(\d+\)\s*/;

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

/**
 * The application holding the playback assertion, or null when only the audio
 * system is named.
 *
 * A named browser is preferred over anything else in the blob: WebCore's
 * assertion is the one that identifies a real source, while the speaker
 * assertion beside it is always coreaudiod's and identifies nobody.
 */
export function parsePlaybackOwner(output) {
	if (typeof output !== "string" || output.length === 0) return null;

	let fallback = null;
	let owner = null;

	for (const line of output.split("\n")) {
		const match = OWNER_LINE.exec(line);
		if (match) {
			owner = match[1].trim();
			if (BROWSER_PLAYBACK_NAME.test(line) && !ANONYMOUS_OWNERS.has(owner)) return owner;
			continue;
		}

		if (!owner || ANONYMOUS_OWNERS.has(owner)) continue;
		const resources = RESOURCES_LINE.exec(line);
		if (!resources) continue;

		const directions = resources[1].split(/\s+/);
		if (directions.includes("audio-out") && !directions.includes("audio-in")) fallback ??= owner;
	}

	return fallback;
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

/**
 * A browser tab title, read as a track.
 *
 * The site name is trimmed off the end because the island already has very
 * little room and "— YouTube" is the least informative part of the line. An
 * empty tab, or one still called after the browser, is not a track.
 */
export function parseTabTitle(raw, application) {
	const parsed = parseTrackReply(raw);
	if (!parsed) return null;

	const title = parsed.title.replace(TAB_TITLE_BADGE, "").replace(TAB_TITLE_SUFFIX, "").trim();
	if (!title || title === application) return null;

	return { title: title.slice(0, 200), artist: "" };
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
 * The Apple event that asks one source what it is playing.
 *
 * Players that can also report where they are in the track return two extra
 * lines. It is one round trip rather than three: every `osascript` call is a
 * process launch and an Apple event round trip, and this runs on a poll.
 */
function trackScript(source) {
	if (source.kind === "browser") {
		return `if application "${source.application}" is running then tell application "${source.application}" to return ${source.track}`;
	}

	const body = source.position && source.duration
		? `${source.track} & "\n" & (${source.position} as text) & "\n" & (${source.duration} as text)`
		: source.track;
	return `if application "${source.application}" is running then tell application "${source.application}" to if player state is ${source.state ?? "playing"} then return ${body}`;
}

/** Seconds from an Apple event number, or null when it is not a usable one. */
function seconds(raw, unit) {
	const parsed = Number.parseFloat(String(raw ?? "").replace(",", "."));
	if (!Number.isFinite(parsed) || parsed < 0) return null;
	const value = unit === "ms" ? parsed / 1000 : parsed;
	// A day is not a track. Anything past it is a misread property, not audio.
	return value > 86_400 ? null : value;
}

/**
 * Position and duration from the two extra lines, or null.
 *
 * Both or neither: a progress bar with a position and no length has nothing to
 * draw against, and one with a length and no position is a bar stuck at zero.
 */
export function parseProgressReply(raw, unit = "s") {
	const lines = String(raw ?? "").split("\n");
	if (lines.length < 4) return null;

	const position = seconds(lines[2], "s");
	const duration = seconds(lines[3], unit);
	if (position === null || duration === null || duration <= 0) return null;

	return { positionSeconds: Math.min(position, duration), durationSeconds: duration };
}

/** The known source an assertion owner corresponds to, or null. */
export function sourceForOwner(owner) {
	if (!owner) return null;
	const normalized = owner.trim().toLowerCase();
	return ALL_SOURCES.find((source) => source.application.toLowerCase() === normalized) ?? null;
}

async function readTrackFrom(source, { execFile }) {
	const reply = await runCommand(execFile, "osascript", ["-e", trackScript(source)]);
	const track = source.kind === "browser"
		? parseTabTitle(reply, source.application)
		: parseTrackReply(reply);
	if (!track) return null;

	const progress = source.position ? parseProgressReply(reply, source.durationUnit) : null;
	return {
		...track,
		source: source.id,
		...(progress ?? {}),
		// The island interpolates between polls, so it needs to know when the
		// position it was given was true rather than only what it was.
		...(progress ? { sampledAt: Date.now(), seekable: source.seekable === true } : {}),
	};
}

/**
 * Current media state.
 *
 * `playing` is authoritative. `owner` is the application making the noise, and
 * is what the island names and focuses when nothing more specific is known.
 * `track` is best effort and stays null whenever the source will not identify
 * itself — a Firefox tab, a game, an app without Automation access.
 *
 * The owner is asked first and alone when it is a source we know: a browser
 * playing a video should not be labelled with whatever Music happens to have
 * paused in the background.
 */
export async function readMediaActivity({
	execFile,
	platform = process.platform,
	runningApplications = null,
} = {}) {
	if (platform !== "darwin" || typeof execFile !== "function") {
		return { playing: false, owner: null, track: null };
	}

	const assertions = await runCommand(execFile, "pmset", ASSERTIONS_ARGV);
	const playing = parseAudioOutputActivity(assertions);
	if (!playing) return { playing: false, owner: null, track: null };

	const owner = parsePlaybackOwner(assertions);
	const ownerSource = sourceForOwner(owner);

	if (ownerSource) {
		const track = await readTrackFrom(ownerSource, { execFile });
		return { playing: true, owner: ownerSource.application, track };
	}

	// Nothing usable in the assertion. Fall back to asking the desktop players
	// that will answer, in order — the historical behaviour, and still right
	// when coreaudiod is the only process named.
	for (const source of TRACK_SOURCES) {
		// Asking a stopped app would launch it, so only running apps are asked.
		if (runningApplications && !runningApplications.includes(source.application)) continue;

		const track = await readTrackFrom(source, { execFile });
		if (track) return { playing: true, owner: source.application, track };
	}

	return { playing: true, owner: owner ?? null, track: null };
}

/** Applications the Apple event probe is allowed to talk to. */
export function trackSourceApplications() {
	return ALL_SOURCES.map((source) => source.application);
}

export { ALL_SOURCES, BROWSER_SOURCES, COMMAND_TIMEOUT_MS, TRACK_SOURCES };

/* --------------------------------------------------------------------------
   Focus
   --------------------------------------------------------------------------
   Clicking the album art or the waveform brings the player forward. Any app
   can be activated by name whether or not it speaks Apple events, so this
   works for the browsers and for the players alike — it is the one thing the
   island can always do with a source it cannot otherwise talk to.
   -------------------------------------------------------------------------- */

/** Application names may be activated. Anything stranger is not run. */
const SAFE_APPLICATION_NAME = /^[\p{L}\p{N} .+_-]{1,64}$/u;

export function isFocusableApplication(value) {
	return typeof value === "string" && SAFE_APPLICATION_NAME.test(value.trim());
}

/**
 * Brings an application forward.
 *
 * Guarded on `is running` like every other script here: clicking the island's
 * waveform must never launch a player, only reveal the one already playing.
 */
export async function focusApplication(application, { execFile, platform = process.platform } = {}) {
	if (platform !== "darwin" || typeof execFile !== "function") return { focused: false };
	if (!isFocusableApplication(application)) return { focused: false };

	const name = application.trim();
	const reply = await runCommand(execFile, "osascript", [
		"-e",
		`if application "${name}" is running then tell application "${name}" to activate`,
	]);
	return { focused: reply !== null };
}

/* --------------------------------------------------------------------------
   Transport
   --------------------------------------------------------------------------
   The horizontal swipe steps a track, and stepping a track needs a player that
   will take the instruction. That is the same short list that will name a
   track, and for the same reason: the general answer, `MRMediaRemoteSendCommand`,
   sits behind an entitlement Apple does not grant.

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

/**
 * Moves the playhead, in seconds from the start of the track.
 *
 * Only the players that reported a position can take one back — a scrubber is
 * drawn from `positionSeconds`, so a source that never supplied one has no bar
 * to drag in the first place.
 */
export async function seekMedia(positionSeconds, {
	execFile,
	platform = process.platform,
	runningApplications = null,
} = {}) {
	if (platform !== "darwin" || typeof execFile !== "function") return { sought: false };

	const target = Number(positionSeconds);
	if (!Number.isFinite(target) || target < 0 || target > 86_400) return { sought: false };
	const rounded = Math.round(target * 1000) / 1000;

	for (const source of TRACK_SOURCES) {
		if (!source.seekable) continue;
		if (runningApplications && !runningApplications.includes(source.application)) continue;

		const script = `if application "${source.application}" is running then tell application "${source.application}" to if player state is ${source.state ?? "playing"} then set player position to ${rounded}`;
		const reply = await runCommand(execFile, "osascript", ["-e", script]);
		if (reply !== null) return { sought: true, source: source.id };
	}

	return { sought: false };
}

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
		const script = `if application "${source.application}" is running then tell application "${source.application}" to if player state is ${source.state ?? "playing"} then ${verb}`;
		const reply = await runCommand(execFile, "osascript", ["-e", script]);
		if (reply !== null) return { sent: true, source: source.id };
	}

	return { sent: false };
}
