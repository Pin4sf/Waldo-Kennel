import { expect, test } from "@playwright/test";

// Board routes intentionally use the Figma shell without titlebar history
// chrome. Exercise the persistent arrows between two standard session routes.
test("titlebar back/forward arrows traverse history", async ({ page }) => {
	await page.goto("/#/projects/ao-demo/sessions/demo-working");
	await expect(page).toHaveURL(/sessions\/demo-working/);

	await page.getByRole("button", { name: "Open Merge README screenshot asset update" }).click();
	await expect(page).toHaveURL(/sessions\/demo-ready/);

	const back = page.getByRole("button", { name: "Go back" });
	const forward = page.getByRole("button", { name: "Go forward" });

	await expect(forward).toBeDisabled();
	await expect(back).toBeEnabled();

	await back.click();
	await expect(page).toHaveURL(/sessions\/demo-working/);

	await expect(forward).toBeEnabled();
	await forward.click();
	await expect(page).toHaveURL(/sessions\/demo-ready/);
});
