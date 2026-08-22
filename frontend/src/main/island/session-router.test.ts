// @vitest-environment node
import { describe, expect, it, vi } from "vitest";
import { createIslandSessionRouter } from "./session-router";

describe("createIslandSessionRouter", () => {
	it("validates against a fresh snapshot and opens the canonical project target", async () => {
		const openSession = vi.fn();
		const getSnapshot = vi.fn(async () => ({
			sessions: [{ id: "session-1", projectId: "project-1" }],
		}));
		const router = createIslandSessionRouter({
			getConnection: () => null,
			fetch: vi.fn(),
			openSession,
			service: { getSnapshot },
		});

		await expect(router.focusSession({ sessionId: "session-1" })).resolves.toBe(true);
		expect(getSnapshot).toHaveBeenCalledWith({ activeOnly: false });
		expect(openSession).toHaveBeenCalledWith({ projectId: "project-1", sessionId: "session-1" });
	});

	it("refuses a missing session or a project mismatch without navigating", async () => {
		const openSession = vi.fn();
		const router = createIslandSessionRouter({
			getConnection: () => null,
			fetch: vi.fn(),
			openSession,
			service: {
				getSnapshot: async () => ({ sessions: [{ id: "session-1", projectId: "project-1" }] }),
			},
		});

		await expect(router.focusSession({ projectId: "other", sessionId: "session-1" })).resolves.toBe(false);
		await expect(router.focusSession({ sessionId: "missing" })).resolves.toBe(false);
		expect(openSession).not.toHaveBeenCalled();
	});

	it("rejects unsafe identifiers before reading the daemon", async () => {
		const getSnapshot = vi.fn();
		const router = createIslandSessionRouter({
			getConnection: () => null,
			fetch: vi.fn(),
			openSession: vi.fn(),
			service: { getSnapshot },
		});

		await expect(router.focusSession({ sessionId: " bad" })).rejects.toThrow(/sessionId is invalid/);
		expect(getSnapshot).not.toHaveBeenCalled();
	});
});
