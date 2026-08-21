import assert from "node:assert/strict";
import test from "node:test";

import { createKennelService, KennelServiceError } from "./kennel-service.mjs";

const RUN_INFO = {
	pid: 42,
	port: 4317,
	startedAt: "2026-08-20T08:00:00Z",
	owner: "persistent",
};

function jsonResponse(body, status = 200) {
	return new Response(JSON.stringify(body), {
		status,
		headers: { "content-type": "application/json" },
	});
}

function noContentResponse() {
	return new Response(null, { status: 204 });
}

function readyResponse(overrides = {}) {
	return jsonResponse({
		status: "ready",
		service: "agent-orchestrator-daemon",
		pid: RUN_INFO.pid,
		...overrides,
	});
}

function conversationResponse(sessionId, activities = []) {
	return jsonResponse({
		conversationId: `conversation-${sessionId}`,
		sessionId,
		mode: "chat",
		controller: "ready",
		latestSequence: activities.length,
		hasMoreBefore: false,
		turns: [],
		messages: [],
		activities,
		settings: {},
	});
}

function harness(routeHandlers = {}, runInfo = RUN_INFO) {
	const calls = [];
	const reads = [];
	const fs = {
		async readFile(file, encoding) {
			reads.push({ file, encoding });
			return JSON.stringify(runInfo);
		},
	};
	const fetch = async (url, init = {}) => {
		const parsed = new URL(url);
		const method = init.method ?? "GET";
		const key = `${method} ${parsed.pathname}${parsed.search}`;
		calls.push({ url, key, init });
		const handler = routeHandlers[key];
		if (!handler) throw new Error(`Unexpected request: ${key}`);
		if (typeof handler === "function") return handler({ url, init, parsed });
		// Real fetch creates a fresh response for every request. Clone static
		// fixtures so repeated readiness/conversation reads behave the same way.
		return typeof handler.clone === "function" ? handler.clone() : handler;
	};
	return {
		calls,
		reads,
		service: createKennelService({ fs, fetch, home: "/Users/tester", timeoutMs: 100 }),
	};
}

function withReady(routes = {}) {
	return { "GET /readyz": readyResponse(), ...routes };
}

async function expectServiceError(promise, code) {
	await assert.rejects(promise, (error) => {
		assert.ok(error instanceof KennelServiceError);
		assert.equal(error.code, code);
		return true;
	});
}

test("attach reads ~/.ao/running.json and verifies the exact loopback daemon identity", async () => {
	const fixture = harness(withReady());

	const daemon = await fixture.service.attach();

	assert.deepEqual(daemon, {
		pid: 42,
		port: 4317,
		startedAt: "2026-08-20T08:00:00Z",
		owner: "persistent",
	});
	assert.deepEqual(fixture.reads, [
		{ file: "/Users/tester/.ao/running.json", encoding: "utf8" },
	]);
	assert.equal(fixture.calls.length, 1);
	assert.equal(fixture.calls[0].url, "http://127.0.0.1:4317/readyz");
	assert.equal(fixture.calls[0].init.redirect, "error");
});

test("an injected runFilePath must be absolute and overrides only daemon discovery", async () => {
	const reads = [];
	const fs = {
		async readFile(file, encoding) {
			reads.push({ file, encoding });
			return JSON.stringify(RUN_INFO);
		},
	};
	const fetch = async () => readyResponse();
	const service = createKennelService({
		fs,
		fetch,
		runFilePath: "/Users/tester/.ao/dev/running.json",
	});

	await service.attach();
	assert.equal(service.runFilePath, "/Users/tester/.ao/dev/running.json");
	assert.deepEqual(reads, [
		{ file: "/Users/tester/.ao/dev/running.json", encoding: "utf8" },
	]);
	assert.throws(
		() => createKennelService({ fs, fetch, runFilePath: ".ao/dev/running.json" }),
		/runFilePath must be an absolute path/,
	);
});

test("attach normalizes missing, malformed, and mismatched running daemon states", async (t) => {
	await t.test("missing run file", async () => {
		const fs = {
			async readFile() {
				const error = new Error("missing");
				error.code = "ENOENT";
				throw error;
			},
		};
		const service = createKennelService({ fs, fetch: assert.fail, home: "/tmp/home" });
		await expectServiceError(service.attach(), "DAEMON_NOT_RUNNING");
	});

	await t.test("invalid port", async () => {
		const fixture = harness(withReady(), { ...RUN_INFO, port: 70_000 });
		await expectServiceError(fixture.service.attach(), "RUN_FILE_INVALID");
		assert.equal(fixture.calls.length, 0);
	});

	await t.test("pid mismatch", async () => {
		const fixture = harness({ "GET /readyz": readyResponse({ pid: 99 }) });
		await expectServiceError(fixture.service.attach(), "DAEMON_IDENTITY_MISMATCH");
	});

	await t.test("not ready", async () => {
		const fixture = harness({ "GET /readyz": readyResponse({ status: "starting" }) });
		await expectServiceError(fixture.service.attach(), "DAEMON_NOT_READY");
	});
});

test("getSnapshot uses only the project, session, notification, and selected conversation reads", async () => {
	const approval = {
		kind: "activity",
		id: "activity-1",
		activityKind: "approval",
		status: "pending",
		requestId: "approval-1",
		summary: "Allow command?",
		detail: {
			reason: "The test runner needs a generated cache directory.",
			command: "npm test -- --update",
			cwd: "/repo",
			decisions: [
				{ id: "allow", label: "Allow once" },
				{ id: "deny", label: "Deny" },
			],
		},
	};
	const fixture = harness(withReady({
		"GET /api/v1/projects": jsonResponse({
			projects: [
				{ id: "kennel", name: "Kennel", path: "/repo", kind: "single_repo", sessionPrefix: "kennel" },
				{ id: "other", name: "Other", path: "/other", kind: "single_repo", sessionPrefix: "other" },
			],
		}),
		"GET /api/v1/sessions?project=kennel&active=true": jsonResponse({
			sessions: [
				{ id: "worker-1", projectId: "kennel", kind: "worker", mode: "chat", status: "needs_input" },
			],
		}),
		"GET /api/v1/notifications?status=unresolved&limit=100": jsonResponse({
			notifications: [
				{ id: "notification-1", projectId: "kennel", sessionId: "worker-1", status: "unread" },
				{ id: "notification-2", projectId: "other", sessionId: "other-1", status: "unread" },
			],
			unreadCount: 2,
			unresolvedCount: 2,
		}),
		"GET /api/v1/sessions/worker-1/conversation": conversationResponse("worker-1", [approval]),
	}));

	const snapshot = await fixture.service.getSnapshot({
		projectId: "kennel",
		conversationSessionIds: ["worker-1", "worker-1"],
	});

	assert.equal(snapshot.project.id, "kennel");
	assert.deepEqual(snapshot.sessions.map((session) => session.id), ["worker-1"]);
	assert.deepEqual(snapshot.notifications.map((notification) => notification.id), ["notification-1"]);
	assert.deepEqual(snapshot.notificationCounts, { unread: 1, unresolved: 1 });
	assert.deepEqual(snapshot.conversations["worker-1"].pending.approvals, [
		{
			activityId: "activity-1",
			requestId: "approval-1",
			summary: "Allow command?",
			decisions: [
				{ id: "allow", label: "Allow once" },
				{ id: "deny", label: "Deny" },
			],
			context: {
				reason: "The test runner needs a generated cache directory.",
				command: "npm test -- --update",
				cwd: "/repo",
				truncated: false,
			},
		},
	]);
	assert.equal("activities" in snapshot.conversations["worker-1"].conversation, false);
	assert.equal("messages" in snapshot.conversations["worker-1"].conversation, false);
	assert.deepEqual(
		fixture.calls.map((call) => call.key).sort(),
		[
			"GET /api/v1/notifications?status=unresolved&limit=100",
			"GET /api/v1/projects",
			"GET /api/v1/sessions/worker-1/conversation",
			"GET /api/v1/sessions?project=kennel&active=true",
			"GET /readyz",
		].sort(),
	);
});

test("getSnapshot can hydrate at most 16 chat conversations that explicitly need input", async () => {
	const sessions = Array.from({ length: 18 }, (_, index) => ({
		id: `pending-${index}`,
		projectId: "kennel",
		kind: "worker",
		mode: "chat",
		status: index === 1 || index === 2 ? "working" : "needs_input",
		activity: index === 1
			? { state: "blocked", lastActivityAt: "2026-08-20T08:00:00Z" }
			: index === 2
				? { state: "waiting_input", lastActivityAt: "2026-08-20T08:00:00Z" }
				: { state: "idle", lastActivityAt: "2026-08-20T08:00:00Z" },
	}));
	sessions.push(
		{ id: "terminal-pending", projectId: "kennel", mode: "tui", status: "needs_input" },
		{ id: "chat-idle", projectId: "kennel", mode: "chat", status: "idle", activity: { state: "idle" } },
	);
	const routes = withReady({
		"GET /api/v1/projects": jsonResponse({ projects: [{ id: "kennel", name: "Kennel" }] }),
		"GET /api/v1/sessions?project=kennel&active=true": jsonResponse({ sessions }),
		"GET /api/v1/notifications?status=unresolved&limit=100": jsonResponse({
			notifications: [], unreadCount: 0, unresolvedCount: 0,
		}),
	});
	for (let index = 0; index < 16; index += 1) {
		routes[`GET /api/v1/sessions/pending-${index}/conversation`] = conversationResponse(`pending-${index}`);
	}
	const fixture = harness(routes);

	const snapshot = await fixture.service.getSnapshot({
		projectId: "kennel",
		includePendingConversations: true,
	});

	assert.deepEqual(
		Object.keys(snapshot.conversations),
		Array.from({ length: 16 }, (_, index) => `pending-${index}`),
	);
	assert.equal(snapshot.pendingConversationsTruncated, true);
	assert.equal(fixture.calls.some((call) => call.url.includes("terminal-pending/conversation")), false);
	assert.equal(fixture.calls.some((call) => call.url.includes("chat-idle/conversation")), false);
	assert.equal(fixture.calls.some((call) => call.url.includes("pending-16/conversation")), false);
});

test("getSnapshot can hydrate active chat conversations when explicitly requested", async () => {
	const fixture = harness(withReady({
		"GET /api/v1/projects": jsonResponse({ projects: [{ id: "kennel", name: "Kennel" }] }),
		"GET /api/v1/sessions?active=true": jsonResponse({
			sessions: [
				{ id: "active-chat", projectId: "kennel", mode: "chat", status: "working", activity: { state: "active" } },
				{ id: "idle-chat", projectId: "kennel", mode: "chat", status: "idle", activity: { state: "idle" } },
				{ id: "active-tui", projectId: "kennel", mode: "tui", status: "working", activity: { state: "active" } },
			],
		}),
		"GET /api/v1/notifications?status=unresolved&limit=100": jsonResponse({
			notifications: [], unreadCount: 0, unresolvedCount: 0,
		}),
		"GET /api/v1/sessions/active-chat/conversation": conversationResponse("active-chat"),
	}));

	const snapshot = await fixture.service.getSnapshot({ includeActiveConversations: true });

	assert.deepEqual(Object.keys(snapshot.conversations), ["active-chat"]);
	assert.equal(snapshot.activeConversationsTruncated, false);
	assert.equal(fixture.calls.some((call) => call.url.includes("idle-chat/conversation")), false);
	assert.equal(fixture.calls.some((call) => call.url.includes("active-tui/conversation")), false);
});

test("getConversation exposes only an allowlisted timeline summary and bounded approval context", async () => {
	const privateTail = "PRIVATE_FULL_COMMAND_TAIL";
	const longCommand = `${"x".repeat(4_096)}${privateTail}`;
	const approval = {
		kind: "activity",
		id: "activity-private",
		activityKind: "approval",
		status: "pending",
		requestId: "approval-private",
		summary: "Allow the requested command?",
		detail: {
			reason: "Required for the selected task.",
			command: longCommand,
			cwd: "/repo",
			decisions: [{ id: "allow", label: "Allow once" }],
		},
	};
	const fixture = harness(withReady({
		"GET /api/v1/sessions/worker-1/conversation": jsonResponse({
			conversationId: "conversation-worker-1",
			sessionId: "worker-1",
			mode: "chat",
			controller: "busy",
			latestSequence: 1,
			hasMoreBefore: false,
			turns: [{ id: "turn-1", state: "running", diff: { patch: privateTail } }],
			messages: [{ id: "message-1", text: privateTail }],
			activities: [approval],
			settings: { model: privateTail },
			capabilities: ["steer", 7],
			account: { planLabel: "Pro", authMode: privateTail },
			rateLimits: {
				planLabel: "Pro",
				title: "5 hour limit",
				primaryUsedPercent: 24,
				primaryResetsInSeconds: 300,
				secondaryUsedPercent: 8,
				secondaryResetsInSeconds: 900,
				private: privateTail,
			},
			usage: {
				cachedTokens: 3,
				contextUsed: 10,
				contextWindow: 100,
				inputTokens: 11,
				outputTokens: 12,
				totalTokens: 23,
				private: privateTail,
			},
		}),
	}));

	const state = await fixture.service.getConversation("worker-1");

	assert.deepEqual(state.conversation, {
		sessionId: "worker-1",
		controller: "busy",
		turns: [{ id: "turn-1", state: "running" }],
		capabilities: ["steer"],
		account: { planLabel: "Pro" },
		rateLimits: {
			planLabel: "Pro",
			title: "5 hour limit",
			primaryUsedPercent: 24,
			primaryResetsInSeconds: 300,
			secondaryUsedPercent: 8,
			secondaryResetsInSeconds: 900,
		},
		usage: {
			cachedTokens: 3,
			contextUsed: 10,
			contextWindow: 100,
			inputTokens: 11,
			outputTokens: 12,
			totalTokens: 23,
		},
	});
	assert.deepEqual(state.pending.approvals, [{
		activityId: "activity-private",
		requestId: "approval-private",
		summary: "Allow the requested command?",
		decisions: [{ id: "allow", label: "Allow once" }],
		context: {
			reason: "Required for the selected task.",
			command: "x".repeat(4_096),
			cwd: "/repo",
			truncated: true,
		},
	}]);
	assert.equal(JSON.stringify(state).includes(privateTail), false);
});

test("getConversation never exposes a partial provider approval decision set", async () => {
	const fixture = harness(withReady({
		"GET /api/v1/sessions/worker-choices/conversation": conversationResponse(
			"worker-choices",
			[{
					kind: "activity",
					activityKind: "approval",
					status: "pending",
					requestId: "approval-choices",
					summary: "Choose an approval action",
					detail: {
						decisions: [
							{ id: "deny", label: "Deny" },
							{ label: "Malformed provider option" },
						],
					},
			}],
		),
	}));

	const state = await fixture.service.getConversation("worker-choices");

	assert.deepEqual(state.pending.approvals[0].decisions, []);
});

test("daemon API errors are normalized without losing the server code or request id", async () => {
	const fixture = harness(withReady({
		"GET /api/v1/projects": jsonResponse({
			error: "internal",
			code: "PROJECT_LIST_FAILED",
			message: "Could not list projects",
			requestId: "request-7",
			details: { source: "registry" },
		}, 500),
		"GET /api/v1/sessions?active=true": jsonResponse({ sessions: [] }),
		"GET /api/v1/notifications?status=unresolved&limit=100": jsonResponse({
			notifications: [], unreadCount: 0, unresolvedCount: 0,
		}),
	}));

	await assert.rejects(fixture.service.getSnapshot(), (error) => {
		assert.ok(error instanceof KennelServiceError);
		assert.equal(error.code, "PROJECT_LIST_FAILED");
		assert.equal(error.status, 500);
		assert.equal(error.retryable, true);
		assert.equal(error.requestId, "request-7");
		assert.deepEqual(error.details, { source: "registry" });
		assert.deepEqual(error.toJSON(), {
			code: "PROJECT_LIST_FAILED",
			message: "Could not list projects",
			status: 500,
			retryable: true,
			details: { source: "registry" },
			requestId: "request-7",
		});
		return true;
	});
});

test("resolveApproval posts only a decision currently offered by the provider", async () => {
	const approval = {
		kind: "activity",
		id: "activity-approval",
		activityKind: "approval",
		status: "pending",
		requestId: "0",
		summary: "Run tests?",
		detail: { decisions: [{ id: "allow-once", label: "Allow once" }] },
	};
	let postedBody;
	const fixture = harness(withReady({
		"GET /api/v1/sessions/worker-1/conversation": conversationResponse("worker-1", [approval]),
		"POST /api/v1/sessions/worker-1/conversation/approvals/0/resolve": ({ init }) => {
			postedBody = JSON.parse(init.body);
			return noContentResponse();
		},
	}));

	const result = await fixture.service.resolveApproval({
		sessionId: "worker-1",
		requestId: "0",
		decisionId: "allow-once",
	});

	assert.deepEqual(result, {
		ok: true,
		sessionId: "worker-1",
		requestId: "0",
		decisionId: "allow-once",
	});
	assert.deepEqual(postedBody, { decisionId: "allow-once" });

	const rejected = harness(withReady({
		"GET /api/v1/sessions/worker-1/conversation": conversationResponse("worker-1", [approval]),
	}));
	await expectServiceError(rejected.service.resolveApproval({
		sessionId: "worker-1",
		requestId: "0",
		decisionId: "always",
	}), "DECISION_NOT_OFFERED");
	assert.equal(rejected.calls.some((call) => call.key.startsWith("POST ")), false);
});

test("resolveInput verifies the pending provider request and exact action/content contract", async () => {
	const inputActivity = {
		kind: "activity",
		id: "activity-input",
		activityKind: "user_input",
		status: "pending",
		requestId: "input-1",
		summary: "Choose a branch",
		detail: {
			inputMode: "form",
			message: "Which branch?",
			schema: { type: "object" },
		},
	};
	let postedBody;
	const fixture = harness(withReady({
		"GET /api/v1/sessions/worker-1/conversation": conversationResponse("worker-1", [inputActivity]),
		"POST /api/v1/sessions/worker-1/conversation/inputs/input-1/resolve": ({ init }) => {
			postedBody = JSON.parse(init.body);
			return noContentResponse();
		},
	}));

	const conversation = await fixture.service.getConversation("worker-1");
	assert.deepEqual(conversation.pending.inputs, [
		{
			activityId: "activity-input",
			requestId: "input-1",
			summary: "Choose a branch",
			detail: {
				inputMode: "form",
				message: "Which branch?",
				schema: { type: "object" },
			},
		},
	]);

	const result = await fixture.service.resolveInput({
		sessionId: "worker-1",
		requestId: "input-1",
		action: "accept",
		content: { branch: "main" },
	});
	assert.deepEqual(result, {
		ok: true,
		sessionId: "worker-1",
		requestId: "input-1",
		action: "accept",
	});
	assert.deepEqual(postedBody, { action: "accept", content: { branch: "main" } });

	await expectServiceError(fixture.service.resolveInput({
		sessionId: "worker-1",
		requestId: "input-1",
		action: "decline",
		content: { branch: "main" },
	}), "INVALID_ARGUMENT");
});

test("steer and interrupt use only their generated chat endpoints", async () => {
	let steerBody;
	let interruptInit;
	const fixture = harness(withReady({
		"POST /api/v1/sessions/worker-1/conversation/steer": ({ init }) => {
			steerBody = JSON.parse(init.body);
			return jsonResponse({ providerTurnId: "provider-turn-7", activityId: "activity-7" }, 202);
		},
		"POST /api/v1/sessions/worker-1/conversation/interrupt": ({ init }) => {
			interruptInit = init;
			return noContentResponse();
		},
	}));

	const steerResult = await fixture.service.steer({
		sessionId: "worker-1",
		text: "Check the failing assertion first.",
		clientMessageId: "island-steer-1",
	});
	assert.deepEqual(steerResult, {
		providerTurnId: "provider-turn-7",
		activityId: "activity-7",
	});
	assert.deepEqual(steerBody, {
		text: "Check the failing assertion first.",
		clientMessageId: "island-steer-1",
	});

	const interruptResult = await fixture.service.interrupt({ sessionId: "worker-1" });
	assert.deepEqual(interruptResult, { ok: true, sessionId: "worker-1" });
	assert.equal(interruptInit.body, undefined);
	assert.equal(interruptInit.headers["content-type"], undefined);
	assert.deepEqual(
		fixture.calls.filter((call) => call.key.startsWith("POST ")).map((call) => call.key),
		[
			"POST /api/v1/sessions/worker-1/conversation/steer",
			"POST /api/v1/sessions/worker-1/conversation/interrupt",
		],
	);

	const rejected = harness(withReady());
	await expectServiceError(rejected.service.steer({ sessionId: "worker-1", text: "   " }), "INVALID_ARGUMENT");
	assert.equal(rejected.calls.length, 0);
});

test("network failures use one stable retryable error shape", async () => {
	const fs = { readFile: async () => JSON.stringify(RUN_INFO) };
	const service = createKennelService({
		fs,
		home: "/Users/tester",
		fetch: async () => {
			throw new TypeError("connection refused");
		},
	});

	await assert.rejects(service.attach(), (error) => {
		assert.ok(error instanceof KennelServiceError);
		assert.equal(error.code, "DAEMON_UNREACHABLE");
		assert.equal(error.retryable, true);
		assert.equal(error.message, "Could not reach the Kennel daemon");
		return true;
	});
});

test("request deadlines normalize to a retryable timeout", async () => {
	const fs = { readFile: async () => JSON.stringify(RUN_INFO) };
	const service = createKennelService({
		fs,
		home: "/Users/tester",
		timeoutMs: 5,
		fetch: async (_url, { signal }) => new Promise((_resolve, reject) => {
			signal.addEventListener("abort", () => {
				const error = new Error("aborted");
				error.name = "AbortError";
				reject(error);
			}, { once: true });
		}),
	});

	await assert.rejects(service.attach(), (error) => {
		assert.ok(error instanceof KennelServiceError);
		assert.equal(error.code, "DAEMON_TIMEOUT");
		assert.equal(error.retryable, true);
		return true;
	});
});
