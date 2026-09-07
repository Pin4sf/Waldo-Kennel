import path from "node:path";
import { fileURLToPath } from "node:url";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const frontendRoot = path.dirname(fileURLToPath(import.meta.url));
const islandRoot = path.resolve(frontendRoot, "../packages/kennel-island");

// The Island remains a shared production renderer and a browser-only visual
// lab in packages/kennel-island. Forge owns only its output directory, keeping
// the detailed Island UI and its renderer tests in one canonical source tree.
export default defineConfig({
	root: islandRoot,
	publicDir: path.join(islandRoot, "public"),
	plugins: [react()],
	build: {
		outDir: path.join(frontendRoot, ".vite/renderer/island_window"),
		emptyOutDir: true,
	},
});
