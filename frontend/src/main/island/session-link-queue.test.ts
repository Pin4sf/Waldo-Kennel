// @vitest-environment node
import { afterEach, describe, expect, it, vi } from "vitest";
import { createSessionLinkQueue } from "./session-link-queue";

async function settle(): Promise<void> {
	for (let index = 0; index < 6; index += 1) await Promise.resolve();
}

afterEach(() => vi.useRealTimers());

describe("createSessionLinkQueue", () => {
	it("keeps a cold-start link until the daemon becomes ready", async () => {
		let ready = false;
		const focusSession = vi.fn(async () => true);
		const queue = createSessionLinkQueue({ isReady: () => ready, focusSession });
		const target = { projectId: "project-1", sessionId: "session-1" };

		queue.enqueue(target);
		await settle();
		expect(focusSession).not.toHaveBeenCalled();
		expect(queue.pending).toEqual(target);

		ready = true;
		queue.readinessChanged();
		await settle();
		expect(focusSession).toHaveBeenCalledWith(target);
		expect(queue.pending).toBeNull();
	});

	it("retries a transient ready-daemon failure without losing the target", async () => {
		vi.useFakeTimers();
		const transient = Object.assign(new Error("starting"), { code: "DAEMON_NOT_READY", retryable: true });
		const focusSession = vi.fn()
			.mockRejectedValueOnce(transient)
			.mockResolvedValueOnce(true);
		const queue = createSessionLinkQueue({
			isReady: () => true,
			focusSession,
			retryInitialMs: 10,
			retryMaxMs: 40,
		});

		queue.enqueue({ projectId: "project-1", sessionId: "session-1" });
		await settle();
		expect(queue.pending).not.toBeNull();
		vi.advanceTimersByTime(10);
		await settle();
		expect(focusSession).toHaveBeenCalledTimes(2);
		expect(queue.pending).toBeNull();
	});

	it("drops an invalid or missing target instead of retrying forever", async () => {
		vi.useFakeTimers();
		const focusSession = vi.fn(async () => false);
		const queue = createSessionLinkQueue({ isReady: () => true, focusSession });

		queue.enqueue({ projectId: "project-1", sessionId: "missing" });
		await settle();
		vi.runAllTimers();
		expect(focusSession).toHaveBeenCalledTimes(1);
		expect(queue.pending).toBeNull();
	});

	it("retains the newest target when another link arrives during a request", async () => {
		let finishFirst!: (value: boolean) => void;
		const firstAttempt = new Promise<boolean>((resolve) => {
			finishFirst = resolve;
		});
		const focusSession = vi.fn()
			.mockImplementationOnce(() => firstAttempt)
			.mockResolvedValueOnce(true);
		const queue = createSessionLinkQueue({ isReady: () => true, focusSession });

		queue.enqueue({ projectId: "project-1", sessionId: "session-1" });
		await settle();
		queue.enqueue({ projectId: "project-2", sessionId: "session-2" });
		finishFirst(true);
		await settle();

		expect(focusSession).toHaveBeenCalledTimes(2);
		expect(focusSession).toHaveBeenLastCalledWith({ projectId: "project-2", sessionId: "session-2" });
		expect(queue.pending).toBeNull();
	});
});
