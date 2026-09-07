import { mkdirSync, writeFileSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import { WebSocket } from "undici";

const port = Number(process.env.CDP_PORT ?? 9_222);
const outputDirectory = process.env.QA_OUTPUT_DIR ?? path.join(os.tmpdir(), "kennel-island-native-qa");
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

async function waitFor(expression, label, timeoutMs = 5_000) {
	const deadline = Date.now() + timeoutMs;
	while (Date.now() < deadline) {
		if (await evaluate(`Boolean(${expression})`)) return;
		await new Promise((resolve) => setTimeout(resolve, 50));
	}
	throw new Error(`Timed out waiting for ${label}`);
}

async function capture(name) {
	// Let AnimatePresence remove the previous transparent layer before asking
	// Chromium for compositor pixels; otherwise the screenshot intentionally
	// catches both sides of the transition.
	await new Promise((resolve) => setTimeout(resolve, 350));
	const result = await call("Page.captureScreenshot", { format: "png", fromSurface: true });
	const filePath = path.join(outputDirectory, `${name}.png`);
	writeFileSync(filePath, Buffer.from(result.data, "base64"));
	return filePath;
}

function rowClickExpression(title) {
	return `(() => {
		const row = [...document.querySelectorAll('.queue-row')]
			.find((candidate) => candidate.querySelector('.queue-row__title')?.textContent === ${JSON.stringify(title)});
		if (!row) return false;
		row.querySelector('.queue-row__action')?.click();
		return true;
	})()`;
}

await call("Runtime.enable");
await call("Page.enable");
await call("Page.bringToFront");

if (process.env.QA_INSPECT_ONLY === "1") {
	const state = await evaluate(`({
		surface: document.querySelector('.kennel-island')?.getAttribute('data-surface'),
		text: document.body.innerText,
		buttons: [...document.querySelectorAll('button')].map((button) => ({
			className: button.className,
			label: button.getAttribute('aria-label'),
			text: button.textContent,
			disabled: button.disabled,
		})),
	})`);
	socket.close();
	process.stdout.write(`${JSON.stringify(state, null, 2)}\n`);
	process.exit(0);
}

const evidence = { screenshots: {}, mutations: [], queue: [] };

await waitFor(`document.querySelector('[aria-label="Kennel island: compact"]')`, "connected compact island");
evidence.compactText = await evaluate("document.body.innerText");
evidence.screenshots.compact = await capture("01-compact");

await evaluate("document.querySelector('.island-header__notch')?.click()");
await waitFor(`document.querySelector('[aria-label="Kennel island: queue"]')`, "work queue");
evidence.queue = await evaluate(`[...document.querySelectorAll('.queue-row')].map((row) => ({
		title: row.querySelector('.queue-row__title')?.textContent,
		action: row.querySelector('.queue-row__action')?.textContent,
		target: row.querySelector('.queue-row__target')?.textContent?.trim(),
	}))`);
const expectedActions = new Map([
	["Review native shell access", "Approve"],
	["Choose deployment target", "Choose"],
	["Stop a stalled agent turn", "Approve"],
	["Review a long command", "Open"],
	["Polish the Kennel Island", "Steer"],
]);
for (const [title, action] of expectedActions) {
	const row = evidence.queue.find((candidate) => candidate.title === title);
	if (row?.action !== action) throw new Error(`Expected ${title} to show ${action}, got ${row?.action}`);
}
evidence.screenshots.queue = await capture("02-queue");

await evaluate(rowClickExpression("Review native shell access"));
await waitFor(`document.querySelector('[aria-label="Kennel island: permission"]')`, "provider approval");
evidence.approvalButtons = await evaluate(`[...document.querySelectorAll('.permission-button strong')].map((node) => node.textContent)`);
if (JSON.stringify(evidence.approvalButtons) !== JSON.stringify(["Deny", "Allow once", "Allow for session"])) {
	throw new Error(`Provider decisions changed: ${JSON.stringify(evidence.approvalButtons)}`);
}
evidence.screenshots.permission = await capture("03-permission");
await evaluate(`[...document.querySelectorAll('.permission-button')]
	.find((button) => button.textContent.includes('Allow once'))?.click()`);
await waitFor(`document.querySelector('[aria-label="Kennel island: queue"]')`, "queue after approval");

await evaluate(rowClickExpression("Choose deployment target"));
await waitFor(`document.querySelector('[aria-label="Kennel island: choice"]')`, "structured input");
evidence.screenshots.choice = await capture("04-choice");
await evaluate(`[...document.querySelectorAll('.choice-option')]
	.find((button) => button.textContent.includes('Production'))?.click()`);
await waitFor(`document.querySelector('[aria-label="Kennel island: queue"]')`, "queue after input");

await evaluate(rowClickExpression("Stop a stalled agent turn"));
await waitFor(`document.querySelector('[aria-label="Kennel island: permission"]')`, "interrupt approval");
evidence.screenshots.interrupt = await capture("05-interrupt");
await evaluate("document.querySelector('[aria-label=\"Interrupt current turn\"]')?.click()");
const interruptTitle = await evaluate("document.querySelector('[aria-label=\"Interrupt current turn\"]')?.title");
if (!interruptTitle?.includes("Click again")) throw new Error("Interrupt did not require confirmation");
await evaluate("document.querySelector('[aria-label=\"Interrupt current turn\"]')?.click()");
await waitFor(`document.querySelector('[aria-label="Kennel island: compact"]')`, "compact island after interrupt");
await waitFor(`document.querySelector('.island-header__notch')`, "settled resting header after interrupt");
await new Promise((resolve) => setTimeout(resolve, 350));

await evaluate("document.querySelector('.island-header__notch')?.click()");
await waitFor(`document.querySelector('[aria-label="Kennel island: queue"]')`, "queue before usage");
await evaluate("document.querySelector('[aria-label^=\"Open Kennel usage\"]')?.click()");
await waitFor(`document.querySelector('[aria-label="Kennel island: usage"]')`, "provider usage");
evidence.usageText = await evaluate("document.body.innerText");
if (!evidence.usageText.includes("Codex Plus") || !evidence.usageText.includes("38%") || !evidence.usageText.includes("71%")) {
	throw new Error(`Usage panel did not show provider limits: ${evidence.usageText}`);
}
evidence.screenshots.usage = await capture("06-usage");
await evaluate("document.querySelector('[aria-label=\"Back\"]')?.click()");
await waitFor(`document.querySelector('[aria-label="Kennel island: compact"]')`, "compact island after usage");
await new Promise((resolve) => setTimeout(resolve, 350));

await evaluate("document.querySelector('.island-header__notch')?.click()");
await waitFor(`document.querySelector('[aria-label="Kennel island: queue"]')`, "queue before steer");
await evaluate(rowClickExpression("Polish the Kennel Island"));
await waitFor(`document.querySelector('[aria-label="Kennel island: steer"]')`, "steer composer");
evidence.screenshots.steer = await capture("07-steer");
await evaluate(`(() => {
	const textarea = document.querySelector('.steer-panel__input');
	const setter = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value').set;
	setter.call(textarea, 'Keep the notch compact and verify the production interaction.');
	textarea.dispatchEvent(new Event('input', { bubbles: true }));
	return textarea.value;
})()`);
await waitFor(`!document.querySelector('.steer-panel__footer button')?.disabled`, "enabled steer submit");
await evaluate("document.querySelector('.steer-panel__footer button')?.click()");
await waitFor(`document.querySelector('[aria-label="Kennel island: compact"]')`, "compact island after steer");
evidence.screenshots.final = await capture("08-final");
evidence.finalText = await evaluate("document.body.innerText");

socket.close();
process.stdout.write(`${JSON.stringify(evidence, null, 2)}\n`);
