import { expect, test } from "@playwright/test";

test("Home Chat and the global rail preserve one Waldo conversation state", async ({ page }) => {
	await page.goto("/#/home/chat");

	await expect(page.getByRole("link", { name: "Chat" })).toHaveAttribute("aria-current", "page");
	await page.getByRole("textbox", { name: "Message Waldo" }).fill("Keep this local draft");
	await page.getByRole("tab", { name: "Activity" }).click();

	await page.getByRole("link", { name: "Today" }).click();
	await page.getByRole("button", { name: "Open Waldo" }).click();
	await expect(page.getByRole("tab", { name: "Activity" })).toHaveAttribute("aria-selected", "true");

	await page.getByRole("tab", { name: "Conversation" }).click();
	await expect(page.getByRole("textbox", { name: "Message Waldo" })).toHaveValue("Keep this local draft");
});

test("Home Chat answers a deterministic preview with source lineage and hands off to Activity", async ({ page }) => {
	await page.goto("/#/home/chat");

	await page.getByRole("textbox", { name: "Message Waldo" }).fill(
		"What changed in the pricing workshop and what still needs me?",
	);
	await page.getByRole("button", { name: "Send preview" }).click();

	await expect(page.getByText(/Two decisions changed/)).toBeVisible();
	await expect(page.getByRole("link", { name: "Pricing decision note" })).toBeVisible();
	await expect(page.getByRole("link", { name: "Workshop calendar event" })).toBeVisible();
	await expect(page.getByText("Local fixture only · no model, provider, send, or save")).toBeVisible();

	await page.getByRole("button", { name: "Review in Activity" }).click();
	await expect(page.getByRole("region", { name: "Specialist run" })).toBeVisible();
	await expect(page.getByRole("complementary", { name: "Run evidence and authority" })).toBeVisible();
	await expect(page.getByRole("button", { name: "Accept" })).toBeVisible();
});

test("narrow Home Chat remains an internally scrolling destination", async ({ page }) => {
	await page.setViewportSize({ width: 760, height: 620 });
	await page.goto("/#/home/chat");

	const chat = page.getByRole("region", { name: "Waldo", exact: true });
	await expect(chat).toBeVisible();
	await expect(page.getByRole("button", { name: "Close Waldo" })).toHaveCount(0);
	await expect
		.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth))
		.toBe(true);
});
