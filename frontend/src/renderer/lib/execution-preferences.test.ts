import { describe, expect, it } from "vitest";
import { resolvePreferredOutcomeAgent } from "./execution-preferences";

describe("resolvePreferredOutcomeAgent", () => {
	it("inherits the exact Project worker when it is supported", () => {
		expect(resolvePreferredOutcomeAgent("claude-code", ["codex", "claude-code", "opencode"])).toBe("claude-code");
	});

	it("keeps an unconfigured Project unconfigured instead of inventing Codex", () => {
		expect(resolvePreferredOutcomeAgent("", ["codex", "claude-code", "opencode"])).toBe("");
	});

	it("does not revive a historical provider that is outside the current supported inventory", () => {
		expect(resolvePreferredOutcomeAgent("legacy-agent", ["codex", "claude-code", "opencode"])).toBe("");
	});
});
