import { expect, test } from "@playwright/test";

// dev:web (VITE_NO_ELECTRON=1) serves lib/mock-data.ts, whose "demo-review-stack"
// session owns three PRs — so the inspector's Reviews tab is
// enabled. Preview mode supplies deterministic review runs while the project
// config still comes from the daemon, so we stub only that route and prove the
// panel renders a real reviewer card (not the empty "no PR yet" placeholder).

test("the Reviews tab renders the reviewer panel for a session that owns PRs", async ({ page }) => {
	await page.route("**/api/v1/projects/kennel-design", (route) =>
		route.fulfill({
			json: {
				status: "ok",
				project: {
					id: "kennel-design",
					kind: "git",
					name: "kennel-design",
					path: "/demo/kennel-design",
					repo: "kennel-design",
					defaultBranch: "main",
					config: { reviewers: [{ harness: "codex" }] },
				},
			},
		}),
	);

	await page.goto("/#/projects/kennel-design");
	await page.getByRole("button", { name: "Show more" }).first().click();
	await page.getByRole("button", { name: "Open Review stacked browser preview flow" }).click();
	await expect(page).toHaveURL(/sessions\/demo-review-stack/);

	const inspector = page.locator("#inspector");
	await expect(inspector).toBeVisible();

	await inspector.getByRole("tab", { name: "Reviews" }).click();
	const prToggle = inspector.getByRole("button", { name: /Browser preview rail renders inside Kennel #319/ });
	await prToggle.click();
	const prCard = inspector.locator("article").filter({ hasText: "#319" });

	// The preview #319 review surfaces its harness, verdict, body, and
	// rerun action — never the empty state, since this session owns a PR.
	await expect(inspector.getByText("No pull request opened yet.")).toHaveCount(0);
	await expect(prCard.getByText("codex", { exact: true })).toBeVisible();
	await expect(prCard.getByText("Changes requested")).toBeVisible();
	await expect(prCard.getByText(/Earlier codex pass asked for tests/)).toBeVisible();
	await expect(inspector.getByRole("button", { name: "Review latest commit" })).toBeVisible();
	await expect(inspector.getByRole("button", { name: "Open terminal" })).toHaveCount(0);
});

test("the Reviews tab shows the empty state for a session with no PRs", async ({ page }) => {
	await page.goto("/#/projects/kennel-design");
	await page.getByRole("button", { name: "Open Build screenshot-ready dashboard data" }).click();
	await expect(page).toHaveURL(/sessions\/demo-working/);

	const inspector = page.locator("#inspector");
	await expect(inspector).toBeVisible();

	await inspector.getByRole("tab", { name: "Reviews" }).click();
	await expect(inspector.getByText("No pull request opened yet.")).toBeVisible();
});
