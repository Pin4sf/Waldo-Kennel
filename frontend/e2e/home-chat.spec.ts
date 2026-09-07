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
	const decisionSource = page.getByRole("link", { name: "Pricing decision note" });
	await expect(decisionSource).toBeVisible();
	await expect(page.getByRole("link", { name: "Workshop calendar event" })).toBeVisible();
	await expect(page.getByText("Local fixture only · no model, provider, send, or save")).toBeVisible();

	await page.getByRole("button", { name: "Review in Activity" }).click();
	await expect(page.getByRole("region", { name: "Specialist run" })).toBeVisible();
	await expect(page.getByRole("complementary", { name: "Run evidence and authority" })).toBeVisible();
	await expect(page.getByRole("button", { name: "Create specialist" })).toBeDisabled();
	await page.getByRole("button", { name: "Accept" }).click();
	await expect(page.getByText("Approved locally").first()).toBeVisible();
	await page.getByRole("button", { name: "Pause specialist" }).click();
	await expect(page.getByText("Paused").first()).toBeVisible();
	await page.getByRole("button", { name: "Resume specialist" }).click();
	await page.getByRole("button", { name: "Stop" }).click();
	await expect(page.getByText("Stopped").first()).toBeVisible();
	await page.getByRole("button", { name: "Retry preview" }).click();
	await expect(page.getByText("Waiting for approval").first()).toBeVisible();
	await page.getByRole("button", { name: "Return to responsibility" }).click();

	await expect(page.getByRole("tab", { name: "Conversation" })).toHaveAttribute("aria-selected", "true");
	await decisionSource.click();
	await expect(page).toHaveURL(/home\/history\?record=pricing-decision-note/);
	await expect(page.getByRole("heading", { name: "Pricing decision note" })).toBeVisible();
	await expect(page.getByText("Pricing workshop note · local fixture")).toBeVisible();
});

test("compact Chat details trap focus, close with Escape, and restore their trigger", async ({ page }) => {
	await page.setViewportSize({ width: 960, height: 700 });
	await page.goto("/#/home/chat");

	const trigger = page.getByRole("button", { name: "Open context details" });
	await trigger.click();
	const dialog = page.getByRole("dialog", { name: "Conversation context details" });
	await expect(dialog).toBeVisible();
	await expect(dialog.getByRole("button", { name: "Back to conversation" })).toBeFocused();
	await page.keyboard.press("Escape");
	await expect(dialog).toHaveCount(0);
	await expect(trigger).toBeFocused();
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
