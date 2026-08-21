// Drives the resting island through its three shapes and captures each one.
//
// The peek only exists under a real pointer, and the pointer is the one input
// a unit test cannot supply: `useStageInteractivity` reads `mousemove` on the
// window, because that is the only signal Electron forwards while the stage is
// click-through. So this walks the shape from outside, through CDP, and reports
// what the island actually measured rather than what the model intended.
//
// Run against a dev island started with `--remote-debugging-port=9222`.

import { mkdirSync, writeFileSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import { WebSocket } from "undici";

const port = Number(process.env.CDP_PORT ?? 9_222);
const outputDirectory = process.env.QA_OUTPUT_DIR
	?? path.join(os.tmpdir(), "kennel-island-peek-qa");
mkdirSync(outputDirectory, { recursive: true });

const targets = await fetch(`http://127.0.0.1:${port}/json/list`).then((response) => response.json());
const target = targets.find((candidate) => candidate.type === "page" && candidate.title === "Kennel Island");
if (!target?.webSocketDebuggerUrl) throw new Error("Kennel Island DevTools target was not found");

const socket = new WebSocket(target.webSocketDebuggerUrl);
await new Promise((resolve, reject) => {
	socket.addEventListener("open", resolve, { once: true });
	socket.addEventListener("error", reject, { once: true });
});

let nextId = 0;
const pending = new Map();
socket.addEventListener("message", (event) => {
	const message = JSON.parse(String(event.data));
	const request = pending.get(message.id);
	if (!request) return;
	pending.delete(message.id);
	if (message.error) request.reject(new Error(message.error.message));
	else request.resolve(message.result);
});

function call(method, params = {}) {
	return new Promise((resolve, reject) => {
		const id = ++nextId;
		pending.set(id, { resolve, reject });
		socket.send(JSON.stringify({ id, method, params }));
	});
}

async function evaluate(expression) {
	const result = await call("Runtime.evaluate", {
		expression,
		awaitPromise: true,
		returnByValue: true,
	});
	if (result.exceptionDetails) {
		throw new Error(result.exceptionDetails.exception?.description ?? result.exceptionDetails.text);
	}
	return result.result.value;
}

const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

async function capture(name) {
	// The size is sprung, so the shape is still moving for a few hundred
	// milliseconds after the state changes. Screenshotting early would capture a
	// frame the user never rests on.
	await wait(650);
	const result = await call("Page.captureScreenshot", { format: "png", fromSurface: true });
	const filePath = path.join(outputDirectory, `${name}.png`);
	writeFileSync(filePath, Buffer.from(result.data, "base64"));
	return filePath;
}

/** What the island currently is, measured from the DOM rather than the model. */
const MEASURE = `(() => {
	const resting = document.querySelector('.island-resting');
	const header = document.querySelector('.island-body--header');
	const box = header?.getBoundingClientRect();
	return {
		shape: resting?.getAttribute('data-shape') ?? null,
		awake: resting?.getAttribute('data-awake') ?? null,
		width: box ? Math.round(box.width) : null,
		height: box ? Math.round(box.height) : null,
		calibrating: Boolean(document.querySelector('.island-calibration')),
	};
})()`;

async function pointerAt(x, y) {
	await call("Input.dispatchMouseEvent", { type: "mouseMoved", x, y, buttons: 0 });
}

await call("Runtime.enable");
await call("Page.enable");

const stage = await evaluate(`(() => {
	const geometry = window.kennelDesktop ? null : null;
	const anchor = document.querySelector('.island-stage__anchor')?.getBoundingClientRect();
	return {
		innerWidth: window.innerWidth,
		anchorCentreX: anchor ? Math.round(anchor.left + anchor.width / 2) : Math.round(window.innerWidth / 2),
	};
})()`);

const report = { stage, shapes: {}, screenshots: {} };

// 1. Dormant: the pointer is nowhere near the notch.
await pointerAt(10, 400);
await wait(250);
report.shapes.dormant = await evaluate(MEASURE);
report.screenshots.dormant = await capture("01-dormant");

// The peek belongs to the quiet island, so a machine that is playing music or
// running a session has nothing to peek from. That is not a failure — but it is
// also not a pass, and reporting it as one would make this script useless.
report.quiet = report.shapes.dormant.shape === "dormant";
if (!report.quiet) {
	report.skipped = "The island is awake (a session is live, or something is playing). "
		+ "The peek only applies to the quiet island; pause media and retry.";
}

// 2. Peek: the pointer settles on the housing.
await pointerAt(stage.anchorCentreX, 12);
await wait(400);
report.shapes.peek = await evaluate(MEASURE);
report.screenshots.peek = await capture("02-peek");

// 3. Calibration outline, which is what the fine tune is aimed at.
await evaluate(`window.kennelDesktop.updateSettings({ appearance: { calibrating: true } })`);
await wait(250);
report.shapes.calibrating = await evaluate(MEASURE);
report.screenshots.calibrating = await capture("03-calibrating");
await evaluate(`window.kennelDesktop.updateSettings({ appearance: { calibrating: false } })`);

// 4. A fine tune reaches the shape without a restart.
await evaluate(`window.kennelDesktop.updateSettings({ notch: { widthOffset: 8, heightOffset: 4 } })`);
await wait(400);
report.shapes.tuned = await evaluate(MEASURE);
report.screenshots.tuned = await capture("04-tuned");
await evaluate(`window.kennelDesktop.updateSettings({ notch: { widthOffset: 0, heightOffset: 0 } })`);

// 5. Back to dormant once the pointer leaves.
await pointerAt(10, 400);
await wait(400);
report.shapes.afterLeave = await evaluate(MEASURE);
report.screenshots.afterLeave = await capture("05-after-leave");

const failures = [];

// These hold whatever the island is doing.
if (!report.shapes.calibrating.calibrating) failures.push("the calibration outline did not appear");
if (!(report.shapes.tuned.width > report.shapes.peek.width)) failures.push("the width fine tune did not reach the shape");

if (report.quiet) {
	if (report.shapes.peek.shape !== "peek") failures.push("a settled pointer did not produce a peek");
	if (!(report.shapes.peek.width > report.shapes.dormant.width)) failures.push("the peek is not wider than the housing");
	if (!(report.shapes.peek.height > report.shapes.dormant.height)) failures.push("the peek is not taller than the housing");
	if (report.shapes.afterLeave.shape !== "dormant") failures.push("the island did not settle back to dormant");
}

report.failures = failures;
socket.close();
process.stdout.write(`${JSON.stringify(report, null, 2)}\n`);
if (failures.length > 0) process.exit(1);
