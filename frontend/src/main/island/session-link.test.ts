// @vitest-environment node
import { describe, expect, it } from "vitest";
import { parseIslandSessionDeepLink } from "./session-link";

describe("parseIslandSessionDeepLink", () => {
	it("parses the namespaced project and session route", () => {
		expect(parseIslandSessionDeepLink("kennel-app://session/project%20one/session-2", "kennel-app")).toEqual({
			projectId: "project one",
			sessionId: "session-2",
		});
	});

	it("never confuses auth callbacks or another protocol for a session", () => {
		expect(parseIslandSessionDeepLink("kennel-app://callback?code=secret", "kennel-app")).toBeNull();
		expect(parseIslandSessionDeepLink("https://session/project/session", "kennel-app")).toBeNull();
	});

	it.each([
		"kennel-app://session/project",
		"kennel-app://session/project/session/extra",
		"kennel-app://session/project/%00session",
		"kennel-app://session/%20/session",
		"not a url",
	])("rejects malformed or unsafe target %s", (url) => {
		expect(parseIslandSessionDeepLink(url, "kennel-app")).toBeNull();
	});
});
