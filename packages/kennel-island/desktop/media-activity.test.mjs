import assert from "node:assert/strict";
import test from "node:test";
import {
	parseAudioOutputActivity,
	parseTrackReply,
	readMediaActivity,
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

	assert.deepEqual(activity, { playing: false, track: null });
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
		playing: false,
		track: null,
	});
	assert.deepEqual(calls, []);
});

test("only Music and Spotify are ever addressed", () => {
	assert.deepEqual(trackSourceApplications(), ["Music", "Spotify"]);
});
