import { rm } from "node:fs/promises";
import path from "node:path";

export default async function afterPack(context) {
	if (context.electronPlatformName !== "darwin") return;
	const appName = `${context.packager.appInfo.productFilename}.app`;
	const defaultApp = path.join(
		context.appOutDir,
		appName,
		"Contents",
		"Resources",
		"default_app.asar",
	);
	await rm(defaultApp, { force: true });
}
