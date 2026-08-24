import { expect, test } from "@playwright/test";

test.use({
	userAgent:
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
});

test("Home and Work mode control is rendered after route-owned drag strips", async ({ page }) => {
	await page.goto("/#/");

	const dragStrip = page.locator(".workspace-topbar-container").first();
	const modeControl = page.getByRole("navigation", { name: "Waldo mode" });
	await expect(dragStrip).toBeVisible();
	await expect(modeControl).toBeVisible();

	const modeFollowsDragStrip = await dragStrip.evaluate((strip, mode) => {
		return Boolean(strip.compareDocumentPosition(mode as Node) & Node.DOCUMENT_POSITION_FOLLOWING);
	}, await modeControl.elementHandle());

	expect(modeFollowsDragStrip).toBe(true);
});

test("Work reserves a separate topbar lane for notifications and Waldo", async ({ page }) => {
	await page.goto("/#/projects/ao-demo");

	const notifications = page.getByRole("button", { name: /notifications/i });
	const waldo = page.getByRole("button", { name: "Open Waldo" });
	await expect(notifications).toBeVisible();
	await expect(waldo).toBeVisible();

	const notificationBox = await notifications.boundingBox();
	const waldoBox = await waldo.boundingBox();
	expect(notificationBox).not.toBeNull();
	expect(waldoBox).not.toBeNull();
	expect(notificationBox!.x + notificationBox!.width).toBeLessThanOrEqual(waldoBox!.x - 6);
});

test("selected Work opens the Work entry destination from a project Board", async ({ page }) => {
	await page.goto("/#/projects/ao-demo");

	const work = page.getByRole("button", { name: "Work", exact: true });
	await expect(work).toBeVisible();
	await expect(work).toHaveCSS("-webkit-app-region", "no-drag");
	await work.click();

	await expect(page).toHaveURL(/#\/work$/);
	await expect(page.getByRole("heading", { name: "What would you like to take on?" })).toBeVisible();
});

test("the centered mode control leaves New terminal pointer-reachable", async ({ page }) => {
	await page.goto("/#/projects/ao-demo/sessions/demo-working");

	const modeControl = page.getByRole("navigation", { name: "Waldo mode" });
	const newTerminal = page.getByRole("button", { name: "New terminal" });
	await expect(modeControl).toBeVisible();
	await expect(newTerminal).toBeVisible();

	await newTerminal.click();
	await expect(page.getByRole("button", { name: /Close terminal/ }).last()).toBeVisible();
});

test("Board, project session, and Outcome lifecycle keep one Work sidebar", async ({ page }) => {
	const workSidebar = page.locator(".figma-board-sidebar");

	await page.goto("/#/projects/ao-demo");
	await expect(workSidebar).toBeVisible();

	await page.goto("/#/projects/ao-demo/sessions/demo-working");
	await expect(workSidebar).toBeVisible();

	await page.goto("/#/work?project=ao-demo");
	await expect(workSidebar).toBeVisible();
});
