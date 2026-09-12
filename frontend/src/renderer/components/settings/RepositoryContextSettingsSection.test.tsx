import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useSettings, useUpdateRepositoryContextLimits } from "../../hooks/useSettings";
import { appI18n } from "../../i18n";
import { RepositoryContextSettingsSection } from "./RepositoryContextSettingsSection";

vi.mock("../../hooks/useSettings", () => ({
	useSettings: vi.fn(),
	useUpdateRepositoryContextLimits: vi.fn(),
}));

const settings = {
	defaultSessionMode: "tui" as const,
	chatHarnesses: ["codex"],
	reasoning: {
		provider: "codex",
		model: "",
		effort: "",
		configured: true,
		ready: true,
		keyConfigured: false,
		verified: false,
	},
	repositoryContext: {
		maxFiles: 64,
		maxBytes: 2_048,
		maxVisited: 0,
		effectiveMaxFiles: 64,
		effectiveMaxBytes: 2_048,
		effectiveMaxVisited: 0,
	},
};

const update = vi.fn();
const resetMutation = vi.fn();

beforeEach(async () => {
	await appI18n.changeLanguage("en");
	update.mockReset();
	resetMutation.mockReset();
	update.mockResolvedValue(settings.repositoryContext);
	vi.mocked(useSettings).mockReturnValue({ settings, isLoading: false, error: undefined });
	vi.mocked(useUpdateRepositoryContextLimits).mockReturnValue({
		update,
		saving: false,
		error: undefined,
		reset: resetMutation,
	});
});

async function openSection() {
	const user = userEvent.setup();
	render(<RepositoryContextSettingsSection />);
	expect(screen.getByLabelText("Maximum files")).not.toBeVisible();
	await user.click(screen.getByText("Advanced repository inspection limits"));
	expect(screen.getByLabelText("Maximum files")).toBeVisible();
	return user;
}

describe("RepositoryContextSettingsSection", () => {
	it("sends only the field the owner changed", async () => {
		const user = await openSection();
		const input = screen.getByLabelText("Maximum files");
		await user.clear(input);
		await user.type(input, "128");
		await user.click(screen.getByRole("button", { name: "Save repository limits" }));

		await waitFor(() => expect(update).toHaveBeenCalledWith({ maxFiles: 128 }));
		expect(screen.getByRole("button", { name: "Saved" })).toBeDisabled();
	});

	it("distinguishes default reset from an explicit uncapped zero", async () => {
		const user = await openSection();
		const maxBytes = screen.getByLabelText("Maximum bytes");
		const maxVisited = screen.getByLabelText("Maximum entries visited");
		await user.clear(maxBytes);
		await user.clear(maxVisited);
		await user.type(maxVisited, "0");
		await user.click(screen.getByRole("button", { name: "Save repository limits" }));

		await waitFor(() => expect(update).toHaveBeenCalledWith({ maxBytes: null, maxVisited: 0 }));
	});

	it("resets all fields to built-in defaults explicitly", async () => {
		const user = await openSection();
		await user.click(screen.getByRole("button", { name: "Reset all to built-in defaults" }));
		await user.click(screen.getByRole("button", { name: "Save repository limits" }));

		await waitFor(() =>
			expect(update).toHaveBeenCalledWith({ maxFiles: null, maxBytes: null, maxVisited: null }),
		);
	});

	it("keeps invalid input visible and blocks persistence", async () => {
		const user = await openSection();
		const input = screen.getByLabelText("Maximum files");
		await user.clear(input);
		await user.type(input, "-1");

		expect(input).toHaveValue(-1);
		expect(screen.getByRole("status")).toHaveTextContent("Use a whole number of zero or more");
		expect(screen.getByRole("button", { name: "Save repository limits" })).toBeDisabled();
		expect(update).not.toHaveBeenCalled();
	});

	it("retains the draft and surfaces daemon errors", async () => {
		vi.mocked(useUpdateRepositoryContextLimits).mockReturnValue({
			update,
			saving: false,
			error: "repository context settings are unavailable",
			reset: resetMutation,
		});
		const user = await openSection();
		const input = screen.getByLabelText("Maximum files");
		await user.clear(input);
		await user.type(input, "96");

		expect(input).toHaveValue(96);
		expect(screen.getByRole("status")).toHaveTextContent("repository context settings are unavailable");
	});
});
