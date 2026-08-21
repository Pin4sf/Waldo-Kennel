import { execFileSync } from "node:child_process";
import {
	mkdtempSync,
	mkdirSync,
	readFileSync,
	readdirSync,
	rmSync,
	writeFileSync,
} from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const source = path.join(root, "build", "icon.png");
const output = path.join(root, "build", "icon.icns");
const temporary = mkdtempSync(path.join(os.tmpdir(), "kennel-island-icon-"));

// PNG-backed modern ICNS representations. The duplicated pixel sizes are
// required because ic11, ic13, and ic14 are Retina variants.
const representations = [
	["icp4", 16],
	["ic11", 32],
	["icp5", 32],
	["ic12", 64],
	["ic07", 128],
	["ic13", 256],
	["ic08", 256],
	["ic14", 512],
	["ic09", 512],
	["ic10", 1024],
];

function icnsChunk(type, png) {
	const header = Buffer.alloc(8);
	header.write(type, 0, 4, "ascii");
	header.writeUInt32BE(8 + png.length, 4);
	return Buffer.concat([header, png]);
}

try {
	const pngs = new Map();
	for (const size of new Set(representations.map(([, representationSize]) => representationSize))) {
		const destination = path.join(temporary, `${size}.png`);
		execFileSync("/usr/bin/sips", [
			"-z", String(size), String(size), source, "--out", destination,
		], { stdio: "ignore" });
		pngs.set(size, readFileSync(destination));
	}

	const chunks = representations.map(([type, size]) => icnsChunk(type, pngs.get(size)));
	const header = Buffer.alloc(8);
	header.write("icns", 0, 4, "ascii");
	header.writeUInt32BE(8 + chunks.reduce((sum, chunk) => sum + chunk.length, 0), 4);
	mkdirSync(path.dirname(output), { recursive: true });
	writeFileSync(output, Buffer.concat([header, ...chunks]));

	const iconset = path.join(temporary, "validate.iconset");
	execFileSync("/usr/bin/iconutil", ["-c", "iconset", output, "-o", iconset], {
		stdio: "ignore",
	});
	const count = readdirSync(iconset).filter((name) => name.endsWith(".png")).length;
	if (count !== 10) throw new Error(`Expected 10 ICNS representations, found ${count}`);
	process.stdout.write(`Wrote build/icon.icns with ${count} representations.\n`);
} finally {
	rmSync(temporary, { recursive: true, force: true });
}
