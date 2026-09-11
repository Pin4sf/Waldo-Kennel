import assert from "node:assert/strict";
import { test } from "vitest";

import { createBuildIdentity } from "./build-identity.mjs";

test("production identity ignores reusable ambient build ids", () => {
	const previous = process.env.KENNEL_BUILD_ID;
	process.env.KENNEL_BUILD_ID = "reused-package-identity";
	try {
		const first = createBuildIdentity();
		const second = createBuildIdentity();
		assert.match(first, /^build-[0-9a-f-]+$/);
		assert.match(second, /^build-[0-9a-f-]+$/);
		assert.notEqual(first, "reused-package-identity");
		assert.notEqual(second, "reused-package-identity");
		assert.notEqual(first, second);
	} finally {
		if (previous === undefined) delete process.env.KENNEL_BUILD_ID;
		else process.env.KENNEL_BUILD_ID = previous;
	}
});

test("each build consumes a fresh generator value", () => {
	const values = ["first", "second"];
	const uuid = () => values.shift();
	assert.notEqual(createBuildIdentity(uuid), createBuildIdentity(uuid));
});
