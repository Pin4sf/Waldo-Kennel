// @vitest-environment node
import { afterEach, describe, expect, it, vi } from "vitest";
import { createIslandEventStream } from "./event-stream";

const encoder = new TextEncoder();

type OpenStream = {
	url: string;
	init: RequestInit;
	push(value: string): void;
	close(): void;
	cancelled: () => boolean;
};

function streamingFetch() {
	const opened: OpenStream[] = [];
	const fetch = vi.fn((input: string, init: RequestInit = {}) => {
		let controller: ReadableStreamDefaultController<Uint8Array>;
		let wasCancelled = false;
		const body = new ReadableStream<Uint8Array>({
			start(value) {
				controller = value;
			},
			cancel() {
				wasCancelled = true;
			},
		});
		const entry: OpenStream = {
			url: String(input),
			init,
			push(value) {
				controller.enqueue(encoder.encode(value));
			},
			close() {
				controller.close();
			},
			cancelled: () => wasCancelled,
		};
		opened.push(entry);
		return Promise.resolve(new Response(body, {
			status: 200,
			headers: { "content-type": "text/event-stream; charset=utf-8" },
		}));
	});
	return { fetch, opened };
}

async function settle(): Promise<void> {
	for (let index = 0; index < 8; index += 1) await Promise.resolve();
}

afterEach(() => {
	vi.useRealTimers();
});

describe("createIslandEventStream", () => {
	it("accepts only a trusted port and opens both fixed loopback SSE routes", async () => {
		const transport = streamingFetch();
		const stream = createIslandEventStream({
			onInvalidate: vi.fn(),
			fetch: transport.fetch,
		});

		for (const invalid of [0, -1, 65_536, 1.5, Number.NaN]) {
			expect(() => stream.rebind(invalid)).toThrow(/port must be an integer/);
		}
		expect(transport.fetch).not.toHaveBeenCalled();

		stream.rebind(4317);
		await settle();

		expect(stream.port).toBe(4317);
		expect(transport.opened.map(({ url }) => url)).toEqual([
			"http://127.0.0.1:4317/api/v1/events",
			"http://127.0.0.1:4317/api/v1/notifications/stream",
		]);
		for (const { init } of transport.opened) {
			expect(init.method).toBe("GET");
			expect(init.redirect).toBe("error");
			expect(init.signal).toBeInstanceOf(AbortSignal);
			expect(new Headers(init.headers).get("accept")).toBe("text/event-stream");
		}

		stream.stop();
	});

	it("coalesces reconnect and split SSE frames from both streams into one invalidation", async () => {
		vi.useFakeTimers();
		const transport = streamingFetch();
		const onInvalidate = vi.fn();
		const stream = createIslandEventStream({
			onInvalidate,
			fetch: transport.fetch,
			invalidateDelayMs: 150,
		});

		stream.rebind(4317);
		await settle();
		transport.opened[0].push("id: 41\r\nevent: session_updated\r\ndata: {\"sessionId\":\"one\"}\r");
		transport.opened[0].push("\n\r\n");
		transport.opened[1].push(": heartbeat\n\nevent: notification_created\ndata: {}\n\n");
		await settle();

		vi.advanceTimersByTime(149);
		expect(onInvalidate).not.toHaveBeenCalled();
		vi.advanceTimersByTime(1);
		expect(onInvalidate).toHaveBeenCalledTimes(1);

		transport.opened[0].push("event: session_updated\ndata: first\n\n");
		transport.opened[1].push("event: notification_resolved\ndata: second\n\n");
		await settle();
		vi.advanceTimersByTime(150);
		expect(onInvalidate).toHaveBeenCalledTimes(2);

		stream.stop();
	});

	it("aborts old readers and binds fresh streams when the trusted port changes", async () => {
		vi.useFakeTimers();
		const transport = streamingFetch();
		const stream = createIslandEventStream({
			onInvalidate: vi.fn(),
			fetch: transport.fetch,
			reconnectInitialMs: 10,
			reconnectMaxMs: 40,
		});

		stream.rebind(4317);
		await settle();
		const old = [...transport.opened];
		stream.rebind(5123);
		await settle();

		expect(old.every(({ init }) => (init.signal as AbortSignal).aborted)).toBe(true);
		expect(old.every(({ cancelled }) => cancelled())).toBe(true);
		expect(transport.opened.slice(2).map(({ url }) => url)).toEqual([
			"http://127.0.0.1:5123/api/v1/events",
			"http://127.0.0.1:5123/api/v1/notifications/stream",
		]);

		vi.advanceTimersByTime(1_000);
		expect(transport.opened).toHaveLength(4);
		stream.stop();
		expect(stream.port).toBeNull();
	});

	it("reconnects with bounded backoff and stops retrying after stop", async () => {
		vi.useFakeTimers();
		const fetch = vi.fn(() => Promise.reject(new Error("offline")));
		const stream = createIslandEventStream({
			onInvalidate: vi.fn(),
			fetch,
			reconnectInitialMs: 10,
			reconnectMaxMs: 40,
		});

		stream.rebind(4317);
		await settle();
		expect(fetch).toHaveBeenCalledTimes(2);

		vi.advanceTimersByTime(9);
		expect(fetch).toHaveBeenCalledTimes(2);
		vi.advanceTimersByTime(1);
		await settle();
		expect(fetch).toHaveBeenCalledTimes(4);

		vi.advanceTimersByTime(19);
		expect(fetch).toHaveBeenCalledTimes(4);
		vi.advanceTimersByTime(1);
		await settle();
		expect(fetch).toHaveBeenCalledTimes(6);

		vi.advanceTimersByTime(39);
		expect(fetch).toHaveBeenCalledTimes(6);
		vi.advanceTimersByTime(1);
		await settle();
		expect(fetch).toHaveBeenCalledTimes(8);

		// The next retry stays capped at 40 ms rather than doubling again.
		vi.advanceTimersByTime(40);
		await settle();
		expect(fetch).toHaveBeenCalledTimes(10);

		stream.stop();
		vi.advanceTimersByTime(1_000);
		await settle();
		expect(fetch).toHaveBeenCalledTimes(10);
	});

	it("resumes the durable CDC stream with Last-Event-ID after it closes", async () => {
		vi.useFakeTimers();
		const transport = streamingFetch();
		const stream = createIslandEventStream({
			onInvalidate: vi.fn(),
			fetch: transport.fetch,
			reconnectInitialMs: 10,
			reconnectMaxMs: 40,
		});

		stream.rebind(4317);
		await settle();
		transport.opened[0].push("id: 73\nevent: session_updated\ndata: {}\n\n");
		await settle();
		transport.opened[0].close();
		await settle();

		vi.advanceTimersByTime(10);
		await settle();
		const eventCalls = transport.opened.filter(({ url }) => url.endsWith("/api/v1/events"));
		expect(eventCalls).toHaveLength(2);
		expect(new Headers(eventCalls[1].init.headers).get("last-event-id")).toBe("73");
		const notificationCalls = transport.opened.filter(({ url }) => url.endsWith("/notifications/stream"));
		expect(notificationCalls).toHaveLength(1);

		stream.stop();
	});
});
