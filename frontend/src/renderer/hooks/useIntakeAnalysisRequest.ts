import { useQuery } from "@tanstack/react-query";

import type { components } from "../../api/schema";
import { apiClient, hasTrustedApiBaseUrl } from "../lib/api-client";
import { usesPreviewWorkspaceData } from "../lib/preview-mode";

export type IntakeAnalysisRequest = components["schemas"]["ControllersIntakeAnalysisRequestResponse"];

export const intakeAnalysisRequestQueryKey = (intakeId: string | undefined) =>
	["intake-analysis-request", intakeId ?? ""] as const;

/**
 * The newest ask for an agent-authored Contract proposal on one intake, or
 * null when no agent was ever asked.
 *
 * "No agent was asked" is the ordinary case, not an error: the daemon answers
 * 404 for it, and every project without a ready analyzer produces its
 * proposal offline. Treating that 404 as absence rather than failure is what
 * lets the surface stay quiet instead of reporting a problem that is not one.
 *
 * While an ask is open it polls, because the answer arrives from a spawned
 * agent over a callback rather than from anything this client did.
 */
export function useIntakeAnalysisRequest(intakeId: string | undefined, { poll }: { poll: boolean }) {
	const query = useQuery({
		queryKey: intakeAnalysisRequestQueryKey(intakeId),
		enabled: Boolean(intakeId) && !usesPreviewWorkspaceData && hasTrustedApiBaseUrl(),
		refetchInterval: poll ? 3_000 : false,
		// Polls while the document is hidden too: the answer arrives from a
		// spawned agent, so pausing when nobody is looking only delays what
		// the person sees when they come back.
		refetchIntervalInBackground: true,
		queryFn: async (): Promise<IntakeAnalysisRequest | null> => {
			const { data, error, response } = await apiClient.GET("/api/v1/intakes/{intakeId}/analysis-request", {
				params: { path: { intakeId: intakeId as string } },
			});
			if (response.status === 404) return null;
			if (error) throw error;
			return data.request;
		},
	});
	return {
		request: query.data ?? undefined,
		loading: query.isLoading,
	};
}

/**
 * Who authored the proposal now on screen.
 *
 * There is no stored "authored by" field, and adding one would duplicate a
 * fact the request already records. A fulfilled ask means its agent's proposal
 * became the current one; anything else — no ask, refused, expired, cancelled
 * — means what is on screen came from the deterministic baseline.
 */
export function proposalProvenance(request: IntakeAnalysisRequest | undefined): {
	kind: "agent" | "offline";
	harness?: string;
} {
	if (request?.status === "fulfilled") return { kind: "agent", harness: request.harness };
	return { kind: "offline" };
}
