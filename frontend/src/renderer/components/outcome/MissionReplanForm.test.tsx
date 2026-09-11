import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";
const { post } = vi.hoisted(() => ({ post: vi.fn() }));
vi.mock("../../lib/api-client", () => ({
	apiClient: { POST: post },
	apiErrorMessage: (error: Error) => error.message,
	apiErrorCode: () => undefined,
}));
import { MissionReplanForm } from "./MissionReplanForm";

it("retains feedback and withholds resubmission after an ambiguous failure", async () => {
	post.mockRejectedValue(new Error("Network response lost"));
	const cache = new QueryClient({ defaultOptions: { queries: { retry: false } } });
	const invalidate = vi.spyOn(cache, "invalidateQueries");
	render(
		<QueryClientProvider client={cache}>
			<MissionReplanForm outcomeId="one" revision={3} />
		</QueryClientProvider>,
	);
	const user = userEvent.setup();
	await user.type(screen.getByRole("textbox"), "Keep the source unchanged");
	await user.click(screen.getByRole("button", { name: "Request revised Plan" }));
	expect(await screen.findByRole("alert")).toHaveTextContent("Network response lost");
	expect(screen.getByRole("textbox")).toHaveValue("Keep the source unchanged");
	expect(screen.getByRole("button", { name: "Request revised Plan" })).toBeDisabled();
	await user.click(screen.getByRole("button", { name: "Refresh facts" }));
	expect(post).toHaveBeenCalledTimes(1);
	expect(post).toHaveBeenCalledWith("/api/v1/outcomes/{outcomeId}/plans/replan", {
		params: { path: { outcomeId: "one" } },
		body: { expectedContractRevision: 3, feedback: "Keep the source unchanged" },
	});
	expect(invalidate).toHaveBeenCalledWith({ queryKey: ["outcome-plan", "one"] });
});
