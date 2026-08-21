// Compiles the two Swift helpers in `desktop/helpers/`.
//
// Both are optional by design. They exist because AppKit knows two things
// Electron will not tell us — where the notch actually is, and how to tap a
// Force Touch trackpad — and neither is worth a native addon in the dependency
// tree or a rebuild on every install.
//
// A machine without a Swift toolchain is therefore reported and skipped, not
// failed: the island falls back to deriving the notch from the menu bar, and to
// no haptics at all.

import { execFileSync } from "node:child_process";
import { chmodSync, existsSync, mkdirSync, statSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const helpers = path.join(root, "desktop", "helpers");

const targets = [
	{ source: "haptics.swift", output: "kennel-haptics" },
	{ source: "notch-metrics.swift", output: "kennel-notch" },
];

function swiftAvailable() {
	try {
		execFileSync("/usr/bin/xcrun", ["--find", "swiftc"], { stdio: "ignore" });
		return true;
	} catch {
		return false;
	}
}

if (process.platform !== "darwin") {
	process.stdout.write("Skipping helpers: macOS only.\n");
} else if (!swiftAvailable()) {
	process.stdout.write(
		"Skipping helpers: no Swift toolchain. The island derives the notch from the menu bar and runs without haptics.\n",
	);
} else {
	mkdirSync(helpers, { recursive: true });

	for (const { source, output } of targets) {
		const sourcePath = path.join(helpers, source);
		const outputPath = path.join(helpers, output);

		if (existsSync(outputPath) && statSync(outputPath).mtimeMs >= statSync(sourcePath).mtimeMs) {
			process.stdout.write(`${output} is up to date.\n`);
			continue;
		}

		execFileSync("/usr/bin/xcrun", [
			"swiftc",
			"-O",
			"-target", "arm64-apple-macos12.0",
			sourcePath,
			"-o", outputPath,
		], { stdio: "inherit" });
		chmodSync(outputPath, 0o755);
		process.stdout.write(`Wrote desktop/helpers/${output}.\n`);
	}
}
