import { expect, test } from "@playwright/test";

// dev:web (VITE_NO_ELECTRON=1) serves lib/mock-data.ts. The kennel-design
// workspace owns a "demo-review-stack" session carrying three PRs:
// #319 open, #320 open, #321 draft, #317 merged — the multi-PR case this suite
// guards across the inspector rail.

test("the inspector rail stacks every PR a session owns, actionable-first", async ({ page }) => {
	await page.goto("/#/projects/kennel-design");
	await page.getByRole("button", { name: "Show more" }).first().click();
	await page.getByRole("button", { name: "Open Review stacked browser preview flow" }).click();
	await expect(page).toHaveURL(/sessions\/demo-review-stack/);

	const inspector = page.locator("#inspector");
	await expect(inspector).toBeVisible();

	// Plural heading reflects the stack size.
	await expect(inspector.getByText("Pull requests (4)")).toBeVisible();

	// One card per PR, ordered actionable-first.
	// Scope to the PR section: the Activity timeline also renders "Opened PR #n".
	const cards = inspector.getByRole("link", { name: /^Open PR #\d+$/ });
	await expect(cards).toHaveText(["PR #319", "PR #320", "PR #321", "PR #317"]);
});
