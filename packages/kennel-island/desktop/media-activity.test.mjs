import assert from "node:assert/strict";
import test from "node:test";
import {
	parseAudioOutputActivity,
	parsePlaybackOwner,
	parseTabTitle,
	parseProgressReply,
	parseTrackReply,
	readMediaActivity,
	seekMedia,
	sourceForOwner,
	trackSourceApplications,
} from "./media-activity.mjs";

const MICROPHONE_ONLY = `Assertion status system-wide:
   PreventUserIdleSystemSleep     1
Listed by owning process:
   pid 409(coreaudiod): [0x0001] 00:20:34 PreventUserIdleSystemSleep named: "com.apple.audio.BuiltInMicrophoneDevice.context.preventuseridlesleep"
	Resources: audio-in BuiltInMicrophoneDevice
`;

const SPEAKER_PLAYBACK = `${MICROPHONE_ONLY}   pid 409(coreaudiod): [0x0002] 00:20:46 PreventUserIdleSystemSleep named: "com.apple.audio.BuiltInSpeakerDevice.context.preventuseridlesleep"
	Resources: audio-out BuiltInSpeakerDevice
`;

// A call runs the speakers exactly like a song does. The microphone alongside
// them is the only thing that tells the two apart.
const VIDEO_CALL = `Listed by owning process:
   pid 409(coreaudiod): [0x0004] 00:25:00 PreventUserIdleSystemSleep named: "com.apple.audio.BuiltInSpeakerDevice.context.preventuseridlesleep"
	Resources: audio-in audio-out BuiltInSpeakerDevice
   pid 409(coreaudiod): [0x0005] 00:25:00 PreventUserIdleSystemSleep named: "com.apple.audio.VPAUAggregateAudioDevice-0x80b95dc40.context.preventuseridlesleep"
	Resources: audio-in audio-out BuiltInMicrophoneDevice
   pid 53746(WhatsApp): [0x0006] 04:19:19 PreventUserIdleSystemSleep named: "cameracaptured-idleSleepPreventionForBWFigCaptureDevice"
`;

const BROWSER_PLAYBACK = `Listed by owning process:
   pid 76096(Safari): [0x0003] 00:03:59 PreventUserIdleDisplaySleep named: "com.apple.WebCore: HTMLMediaElement playback"
`;

// WebKit keeps this one while a tab merely has media loaded, paused included.
const BROWSER_MEDIA_LOADED = `Listed by owning process:
   pid 404(runningboardd): [0x0007] 00:20:12 PreventUserIdleSystemSleep named: "app<application.com.apple.Safari(501)>404-76096:WebKit Media Playback"
`;

function execFileStub(replies) {
	const calls = [];
	const execFile = (command, argv, _options, callback) => {
		calls.push([command, ...argv]);
		const key = command === "pmset" ? "pmset" : String(argv[1]);
		const match = Object.entries(replies).find(([pattern]) => key.includes(pattern));
		if (!match) {
			callback(new Error("no reply configured"), "");
			return;
		}
		callback(null, match[1]);
	};
	return { execFile, calls };
}

test("a microphone held open by a call is not media playing", () => {
	// Every video call holds audio-in. Treating that as playback would light
	// the waveform through every meeting.
	assert.equal(parseAudioOutputActivity(MICROPHONE_ONLY), false);
});

test("audio leaving the speakers is media playing", () => {
	assert.equal(parseAudioOutputActivity(SPEAKER_PLAYBACK), true);
});

test("a browser actually playing an element counts", () => {
	assert.equal(parseAudioOutputActivity(BROWSER_PLAYBACK), true);
});

test("a tab with media merely loaded does not count", () => {
	assert.equal(parseAudioOutputActivity(BROWSER_MEDIA_LOADED), false);
});

test("a video call is not media, however loud it is", () => {
	assert.equal(parseAudioOutputActivity(VIDEO_CALL), false);
});

test("empty or unreadable assertion output is silence, not a crash", () => {
	assert.equal(parseAudioOutputActivity(""), false);
	assert.equal(parseAudioOutputActivity(null), false);
	assert.equal(parseAudioOutputActivity(undefined), false);
});

test("a two-line reply becomes a title and an artist", () => {
	assert.deepEqual(parseTrackReply("Challenge (feat. Juice WRLD)\nYoung Thug"), {
		title: "Challenge (feat. Juice WRLD)",
		artist: "Young Thug",
	});
});

test("a reply without a title yields no track", () => {
	assert.equal(parseTrackReply(""), null);
	assert.equal(parseTrackReply("\nYoung Thug"), null);
	assert.equal(parseTrackReply(null), null);
});

test("a titled track is read from a running player", async () => {
	const { execFile, calls } = execFileStub({
		pmset: SPEAKER_PLAYBACK,
		Spotify: "Challenge (feat. Juice WRLD)\nYoung Thug",
	});

	const activity = await readMediaActivity({
		execFile,
		platform: "darwin",
		runningApplications: ["Spotify"],
	});

	assert.equal(activity.playing, true);
	assert.equal(activity.track.title, "Challenge (feat. Juice WRLD)");
	assert.equal(activity.track.artist, "Young Thug");
	// Music is not running, so it is never asked and never launched.
	assert.equal(calls.some((call) => call.join(" ").includes("Music")), false);
});

test("browser audio stays playing with no invented title", async () => {
	const { execFile } = execFileStub({ pmset: BROWSER_PLAYBACK });

	const activity = await readMediaActivity({
		execFile,
		platform: "darwin",
		runningApplications: [],
	});

	assert.equal(activity.playing, true);
	assert.equal(activity.track, null);
});

test("silence skips the Apple event probe entirely", async () => {
	const { execFile, calls } = execFileStub({ pmset: MICROPHONE_ONLY });

	const activity = await readMediaActivity({
		execFile,
		platform: "darwin",
		runningApplications: ["Spotify", "Music"],
	});

	assert.deepEqual(activity, { playing: false, owner: null, track: null });
	assert.deepEqual(calls, [["pmset", "-g", "assertions"]]);
});

test("a player that refuses automation leaves the track unknown", async () => {
	// Declining the Automation prompt makes osascript fail. The waveform still
	// belongs on screen; the title does not.
	const { execFile } = execFileStub({ pmset: SPEAKER_PLAYBACK });

	const activity = await readMediaActivity({
		execFile,
		platform: "darwin",
		runningApplications: ["Spotify"],
	});

	assert.equal(activity.playing, true);
	assert.equal(activity.track, null);
});

test("non-macOS hosts report silence without shelling out", async () => {
	const { execFile, calls } = execFileStub({ pmset: SPEAKER_PLAYBACK });

	assert.deepEqual(await readMediaActivity({ execFile, platform: "linux" }), {
		owner: null,
		playing: false,
		track: null,
	});
	assert.deepEqual(calls, []);
});

test("only named players and browsers are ever addressed", () => {
	const applications = trackSourceApplications();
	assert.ok(applications.includes("Music"));
	assert.ok(applications.includes("Spotify"));
	assert.ok(applications.includes("Safari"));
	assert.ok(applications.includes("Google Chrome"));
	// Firefox ships no Apple event vocabulary, so there is nothing to ask it.
	assert.equal(applications.includes("Firefox"), false);
	assert.equal(new Set(applications).size, applications.length);
});

test("the browser holding the playback assertion is the owner", () => {
	assert.equal(parsePlaybackOwner(BROWSER_PLAYBACK), "Safari");
});

test("the audio system is nobody, so the owner stays unknown", () => {
	assert.equal(parsePlaybackOwner(SPEAKER_PLAYBACK), null);
	assert.equal(parsePlaybackOwner(""), null);
	assert.equal(parsePlaybackOwner(null), null);
});

test("an owner we know maps to the source that can be asked", () => {
	assert.equal(sourceForOwner("Safari")?.kind, "browser");
	assert.equal(sourceForOwner("Spotify")?.kind, "player");
	assert.equal(sourceForOwner("Firefox"), null);
	assert.equal(sourceForOwner(null), null);
});

test("a tab title loses its site name and its unread badge", () => {
	assert.deepEqual(parseTabTitle("(3) Mannequin Challenge - YouTube", "Google Chrome"), {
		title: "Mannequin Challenge",
		artist: "",
	});
	assert.deepEqual(parseTabTitle("Live set — SoundCloud", "Safari"), {
		title: "Live set",
		artist: "",
	});
});

test("a tab still named after its browser is not a track", () => {
	assert.equal(parseTabTitle("Safari", "Safari"), null);
	assert.equal(parseTabTitle("- YouTube", "Google Chrome"), null);
	assert.equal(parseTabTitle("", "Safari"), null);
});

test("a browser tab that names itself becomes the track", async () => {
	const { execFile, calls } = execFileStub({
		pmset: BROWSER_PLAYBACK,
		Safari: "Mannequin Challenge - YouTube\n",
	});

	const activity = await readMediaActivity({ execFile, platform: "darwin" });

	assert.equal(activity.playing, true);
	assert.equal(activity.owner, "Safari");
	assert.equal(activity.track.title, "Mannequin Challenge");
	// The owner is asked and nobody else: a paused Spotify in the background
	// must not label a video playing in Safari.
	assert.equal(calls.some((call) => call.join(" ").includes("Spotify")), false);
});

test("position and duration are read from the same reply as the title", () => {
	assert.deepEqual(parseProgressReply("Song\nArtist\n61.5\n245.0"), {
		positionSeconds: 61.5,
		durationSeconds: 245,
	});
});

test("Spotify's milliseconds become seconds, Music's do not", () => {
	assert.equal(parseProgressReply("Song\nArtist\n61\n245000", "ms")?.durationSeconds, 245);
	assert.equal(parseProgressReply("Song\nArtist\n61\n245", "s")?.durationSeconds, 245);
});

test("a reply missing either number has no progress to draw", () => {
	assert.equal(parseProgressReply("Song\nArtist"), null);
	assert.equal(parseProgressReply("Song\nArtist\n61\n0"), null);
	assert.equal(parseProgressReply("Song\nArtist\nmissing\n245"), null);
});

test("a position past the end is clamped rather than overrunning the bar", () => {
	assert.equal(parseProgressReply("Song\nArtist\n900\n245")?.positionSeconds, 245);
});

test("seeking refuses a position no track could have", async () => {
	const { execFile, calls } = execFileStub({ Music: "" });
	assert.deepEqual(await seekMedia(Number.NaN, { execFile, platform: "darwin" }), { sought: false });
	assert.deepEqual(await seekMedia(-4, { execFile, platform: "darwin" }), { sought: false });
	assert.deepEqual(calls, []);
});
