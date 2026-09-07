import { expect, test } from "@playwright/test";
import { installFakeAgent } from "./support/fake-bridge";

const projectId = "model-scroll-proj";
const models = Array.from({ length: 30 }, (_, index) => ({
	id: `codex/model-${index + 1}`,
	label: `codex/model-${index + 1}`,
	provider: "codex",
	isDefault: index === 0,
}));

test("renderer: hidden model-menu scrollbar keeps wheel scrolling functional @T0", async ({ page }) => {
	await installFakeAgent(page, {
		projectId,
		projectName: projectId,
		workers: [{ id: "model-worker", title: "Model selection worker" }],
	});
	await page.route("http://127.0.0.1:8080/api/v1/**", async (route) => {
		const request = route.request();
		const pathname = new URL(request.url()).pathname;
		if (pathname === "/api/v1/agents" || pathname === "/api/v1/agents/refresh") {
			await route.fulfill({
				json: {
					supported: [{ id: "codex", label: "Codex" }],
					installed: [{ id: "codex", label: "Codex", authStatus: "authorized" }],
					authorized: [{ id: "codex", label: "Codex", authStatus: "authorized" }],
				},
			});
			return;
		}
		if (pathname === `/api/v1/projects/${projectId}`) {
			await route.fulfill({
				json: {
					status: "ok",
					project: {
						id: projectId,
						agent: "codex",
						config: { worker: { agent: "codex" } },
					},
				},
			});
			return;
		}
		if (pathname === "/api/v1/agents/codex/models") {
			await route.fulfill({
				json: {
					agent: "codex",
					selectionMode: "catalog",
					models,
					allowCustom: true,
					refreshRecommended: false,
				},
			});
			return;
		}
		await route.fulfill({ json: { status: "ok" } });
	});

	await page.goto(`/#/projects/${projectId}`);
	await page.getByRole("button", { name: projectId, exact: true }).click({ button: "right" });
	await page.getByRole("menuitem", { name: "New session" }).click();
	const dialog = page.getByRole("dialog", { name: "Define an outcome" });
	await expect(dialog).toBeVisible();
	await dialog.getByRole("button", { name: "Model" }).click();

	const scroller = page.locator(".model-menu-scroll");
	await expect(scroller).toBeVisible();
	const geometry = await scroller.evaluate((element) => ({
		clientHeight: element.clientHeight,
		scrollHeight: element.scrollHeight,
	}));
	expect(geometry.scrollHeight).toBeGreaterThan(geometry.clientHeight);
	await scroller.hover();
	await page.mouse.wheel(0, 400);

	await expect.poll(() => scroller.evaluate((element) => element.scrollTop)).toBeGreaterThan(0);
});
