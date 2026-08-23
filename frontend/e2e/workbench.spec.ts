import { expect, test } from "@playwright/test";

// The Playwright web server runs `dev:web` (VITE_NO_ELECTRON=1), so
// useWorkspaceQuery serves the deterministic preview fixtures from
// lib/mock-data.ts instead of hitting a daemon. The tests run in Chromium
// (no window.kennel), so the terminal shows its browser-preview surface.

test("renders the current Work project and session shell", async ({ page }) => {
	await page.goto("/");
	await expect(page.getByRole("navigation", { name: "Waldo mode" })).toBeVisible();
	await expect(page.getByText("Projects")).toBeVisible();
	await expect(page.getByRole("button", { name: "Open Build screenshot-ready dashboard data" })).toBeVisible();
	await expect(page.getByTestId("board")).toBeVisible();
});

test("deep-links into a worker session", async ({ page }) => {
	await page.goto("/#/projects/ao-demo/sessions/demo-working");
	await expect(page.getByTestId("session-detail")).toBeVisible();
	await expect(page.getByRole("tab", { name: "Summary" })).toBeVisible();
	await expect(page.getByRole("button", { name: "New terminal" })).toBeVisible();
});

test("drilling into a worker opens its inspectable summary rail", async ({ page }) => {
	await page.goto("/");
	await page.getByRole("button", { name: "Open Build screenshot-ready dashboard data" }).click();
	await expect(page.getByTestId("session-detail")).toBeVisible();
	await expect(page.getByRole("tab", { name: "Summary" })).toHaveAttribute("aria-selected", "true");
	await expect(page.getByText("Activity", { exact: true })).toBeVisible();
	await expect(page.getByText("Working", { exact: true })).toBeVisible();
});
