import { expect, test } from "@playwright/test";

test.use({
	userAgent:
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
});

test("Home and Work mode control is owned by the sidebar", async ({ page }) => {
	await page.goto("/#/");

	const sidebar = page.locator('[data-slot="sidebar"]');
	const modeControl = page.getByRole("navigation", { name: "Waldo mode" });
	await expect(sidebar).toBeVisible();
	await expect(modeControl).toBeVisible();

	await expect(sidebar.getByRole("navigation", { name: "Waldo mode" })).toHaveCount(1);
	await expect(page.locator("main").getByRole("navigation", { name: "Waldo mode" })).toHaveCount(0);
});

test("preview Work entry cannot register a real project", async ({ page }) => {
	const posts: string[] = [];
	page.on("request", (request) => {
		if (request.method() === "POST") posts.push(request.url());
	});

	await page.goto("/#/work");
	await page.getByRole("button", { name: "Start with Work" }).click();
	await expect(page.getByRole("heading", { name: "Select a project" })).toBeVisible();
	await expect(page.getByRole("button", { name: "Add a project" })).toHaveCount(0);
	expect(posts).toEqual([]);
});

test("Work reserves a separate topbar lane for notifications and Waldo", async ({ page }) => {
	await page.goto("/#/projects/kennel-design");

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

test("the sidebar mode control leaves New terminal pointer-reachable", async ({ page }) => {
	await page.goto("/#/projects/kennel-design/sessions/demo-working");

	const modeControl = page.getByRole("navigation", { name: "Waldo mode" });
	const newTerminal = page.getByRole("button", { name: "New terminal" });
	await expect(modeControl).toBeVisible();
	await expect(newTerminal).toBeVisible();

	await newTerminal.click();
	await expect(page.getByRole("button", { name: /Close terminal/ }).last()).toBeVisible();
});

test("Board, project session, and Outcome lifecycle keep one Work sidebar", async ({ page }) => {
	const workSidebar = page.locator(".figma-board-sidebar");

	await page.goto("/#/projects/kennel-design");
	await expect(workSidebar).toBeVisible();

	await page.goto("/#/projects/kennel-design/sessions/demo-working");
	await expect(workSidebar).toBeVisible();

	await page.goto("/#/work?project=kennel-design");
	await expect(workSidebar).toBeVisible();
});
