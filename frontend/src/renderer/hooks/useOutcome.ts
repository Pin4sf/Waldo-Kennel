/**
 * Outcome contract reads and writes for the Understand stage.
 *
 * The renderer is a thin surface over the daemon's canonical Outcome API: it
 * never derives contract state locally, never writes SQLite, and never claims
 * a save the daemon has not answered. Every mutation returns the authoritative
 * OutcomeEnvelope, and the caller renders from that response — there are no
 * optimistic Outcome claims here.
 */

import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import type { components } from "../../api/schema";
import { apiClient, apiErrorCode, apiErrorMessage } from "../lib/api-client";

export type OutcomeRecord = components["schemas"]["OutcomeResponse"];
export type ContractRevisionRecord = components["schemas"]["ContractRevisionResponse"];
export type CreateOutcomeRequest = components["schemas"]["CreateOutcomeRequest"];
export type ReviseOutcomeContractRequest = components["schemas"]["ReviseOutcomeContractRequest"];

type OutcomeEnvelope = components["schemas"]["OutcomeEnvelope"];
type ApiErrorBody = components["schemas"]["APIError"];

export function outcomeQueryKey(outcomeId: string | undefined) {
	return ["outcome", outcomeId ?? ""] as const;
}

/** The daemon's typed refusal when the Outcome does not exist. */
export const OUTCOME_NOT_FOUND = "OUTCOME_NOT_FOUND";
/** The daemon's typed refusal when a revision races another writer. */
export const OUTCOME_CONTRACT_CONFLICT = "OUTCOME_CONTRACT_CONFLICT";

/**
 * Codes that describe a permanently unactionable request: retrying the same
 * bytes cannot help, so the query/mutation must surface them instead of
 * spinning. Everything else is treated as transient.
 */
const PERMANENT_CODES = new Set([
	OUTCOME_NOT_FOUND,
	"PROJECT_NOT_FOUND",
	"PROJECT_REQUIRED",
	"REQUEST_KEY_REQUIRED",
	"EXPECTED_REVISION_REQUIRED",
	"OUTCOME_TITLE_REQUIRED",
	"OUTCOME_TITLE_TOO_LONG",
	"OUTCOME_GOAL_REQUIRED",
	"OUTCOME_CRITERIA_REQUIRED",
	"OUTCOME_REVIEW_REQUIRED",
	"PLAN_ID_REQUIRED",
	"PLAN_CAPABILITY_MISSING",
]);

/**
 * Classify a mutation/query failure for the Understand surface.
 *
 * - `conflict` — the daemon answered 409 OUTCOME_CONTRACT_CONFLICT; the
 *   details carry the expected/current revision numbers.
 * - `permanent` — a typed validation refusal; retrying cannot help.
 * - `offline` — the request never reached the daemon (no trusted base URL or
 *   a transport-level throw), so there is no code at all.
 * - otherwise `retryable` — the daemon (or network) failed transiently.
 */
export interface OutcomeFailure {
	kind: "conflict" | "permanent" | "offline" | "retryable";
	code?: string;
	message: string;
	expectedRevision?: number;
	currentRevision?: number;
}

function readRevisionDetail(details: unknown, key: string): number | undefined {
	if (typeof details !== "object" || details === null) return undefined;
	const value = (details as Record<string, unknown>)[key];
	return typeof value === "number" ? value : undefined;
}

export function classifyOutcomeFailure(error: unknown): OutcomeFailure {
	const code = apiErrorCode(error);
	const message = apiErrorMessage(error);
	if (code === OUTCOME_CONTRACT_CONFLICT) {
		const details = (typeof error === "object" && error !== null ? (error as ApiErrorBody).details : undefined);
		return {
			kind: "conflict",
			code,
			message,
			expectedRevision: readRevisionDetail(details, "expectedRevision"),
			currentRevision: readRevisionDetail(details, "currentRevision"),
		};
	}
	if (code && PERMANENT_CODES.has(code)) {
		return { kind: "permanent", code, message };
	}
	if (!code) {
		// Every daemon error envelope carries a code. No code means the request
		// never reached the daemon: untrusted base URL (synthetic 503) or a
		// transport failure.
		return { kind: "offline", message };
	}
	return { kind: "retryable", code, message };
}

export interface OutcomeQueryResult {
	outcome?: OutcomeRecord;
	isLoading: boolean;
	/** Set when the daemon answers that this Outcome does not exist. */
	unavailable?: { code: string; message: string };
	failure?: OutcomeFailure;
	refetch: () => void;
}

async function fetchOutcomeRecord(outcomeId: string): Promise<OutcomeRecord> {
	const { data, error } = await apiClient.GET("/api/v1/outcomes/{outcomeId}", {
		params: { path: { outcomeId } },
	});
	if (error) throw error;
	return (data as OutcomeEnvelope).outcome;
}

/**
 * Read the current authoritative Outcome through the shared cache, e.g. after
 * a conflict tells the surface its view is stale.
 */
export function refetchOutcome(queryClient: QueryClient, outcomeId: string): Promise<OutcomeRecord> {
	return queryClient.fetchQuery({
		queryKey: outcomeQueryKey(outcomeId),
		queryFn: () => fetchOutcomeRecord(outcomeId),
	});
}

export function useOutcome(outcomeId: string | undefined): OutcomeQueryResult {
	const query = useQuery({
		queryKey: outcomeQueryKey(outcomeId),
		enabled: Boolean(outcomeId),
		queryFn: () => fetchOutcomeRecord(outcomeId as string),
		retry: (attempt, error) => {
			const code = apiErrorCode(error);
			if (code && PERMANENT_CODES.has(code)) return false;
			return attempt < 2;
		},
	});

	if (query.error) {
		const failure = classifyOutcomeFailure(query.error);
		return {
			isLoading: query.isLoading,
			failure,
			refetch: () => {
				void query.refetch();
			},
			unavailable:
				failure.kind === "permanent"
					? { code: failure.code ?? "", message: failure.message }
					: undefined,
		};
	}

	return {
		outcome: query.data,
		isLoading: query.isLoading,
		refetch: () => {
			void query.refetch();
		},
	};
}

export interface OutcomeWriteState<TInput> {
	/** The daemon has accepted the write and returned the authoritative Outcome. */
	succeeded: boolean;
	pending: boolean;
	failure?: OutcomeFailure;
	reset: () => void;
	save: (input: TInput) => Promise<OutcomeRecord>;
}

function useOutcomeWrite<TInput>(mutationFn: (input: TInput) => Promise<OutcomeRecord>): OutcomeWriteState<TInput> {
	const queryClient = useQueryClient();
	const mutation = useMutation({
		mutationFn,
		// Cache the authoritative answer so a remounted surface reads the saved
		// contract instead of re-asking, and refetch nothing optimistically.
		onSuccess: (outcome) => {
			queryClient.setQueryData(outcomeQueryKey(outcome.id), outcome);
		},
	});

	return {
		pending: mutation.isPending,
		failure: mutation.error ? classifyOutcomeFailure(mutation.error) : undefined,
		reset: () => mutation.reset(),
		/** Runs the write and resolves with the daemon's authoritative Outcome. */
		save: mutation.mutateAsync,
		succeeded: mutation.isSuccess,
	};
}

/**
 * Create the Outcome with its first immutable ContractRevision.
 *
 * The caller supplies the idempotency `requestKey`; replaying the same key
 * after an ambiguous failure returns the original Outcome instead of writing
 * twice, which is why retries must reuse the key rather than mint a new one.
 */
export function useCreateOutcome(projectId: string | undefined) {
	return useOutcomeWrite(async (input: CreateOutcomeRequest) => {
		const { data, error } = await apiClient.POST("/api/v1/projects/{id}/outcomes", {
			params: { path: { id: projectId as string } },
			body: input,
		});
		if (error) throw error;
		return (data as OutcomeEnvelope).outcome;
	});
}

/**
 * Append the next immutable ContractRevision. `expectedRevision` must name
 * the current revision; a stale pointer comes back as a typed conflict with
 * both revision numbers in the details.
 */
export function useReviseOutcomeContract(outcomeId: string | undefined) {
	return useOutcomeWrite(async (input: ReviseOutcomeContractRequest) => {
		const { data, error } = await apiClient.POST("/api/v1/outcomes/{outcomeId}/revisions", {
			params: { path: { outcomeId: outcomeId as string } },
			body: input,
		});
		if (error) throw error;
		return (data as OutcomeEnvelope).outcome;
	});
}

/* ----------------------------- Plans (#26) ------------------------------ */

export type PlanRecord = components["schemas"]["PlanRevisionResponse"];
type PlanEnvelope = components["schemas"]["PlanEnvelope"];

export const PLAN_NOT_FOUND = "PLAN_NOT_FOUND";
/** The plan binds a superseded contract revision: a fresh brief is required. */
export const PLAN_CONTRACT_STALE = "PLAN_CONTRACT_STALE";
/** Authority layers no longer allow every capability the plan freezes. */
export const PLAN_CAPABILITY_UNAUTHORIZED = "PLAN_CAPABILITY_UNAUTHORIZED";

function planQueryKey(outcomeId: string | undefined) {
	return ["outcome-plan", outcomeId ?? ""] as const;
}

async function fetchLatestPlan(outcomeId: string): Promise<PlanRecord> {
	const { data, error } = await apiClient.GET("/api/v1/outcomes/{outcomeId}/plan", {
		params: { path: { outcomeId } },
	});
	if (error) throw error;
	return (data as PlanEnvelope).plan;
}

export interface OutcomePlanQueryResult {
	/** Undefined when no plan exists yet — absence is an answer, not an error. */
	plan?: PlanRecord;
	isLoading: boolean;
	failure?: OutcomeFailure;
	refetch: () => void;
}

/**
 * The newest plan of any status for one Outcome.
 */
export function useOutcomePlan(outcomeId: string | undefined): OutcomePlanQueryResult {
	const query = useQuery({
		queryKey: planQueryKey(outcomeId),
		enabled: Boolean(outcomeId),
		queryFn: () => fetchLatestPlan(outcomeId as string),
		retry: (attempt, error) => {
			const code = apiErrorCode(error);
			if (code === PLAN_NOT_FOUND || code === OUTCOME_NOT_FOUND) return false;
			return attempt < 2;
		},
	});

	if (query.error) {
		if (apiErrorCode(query.error) === PLAN_NOT_FOUND) {
			return { isLoading: false, refetch: () => void query.refetch() };
		}
		return { isLoading: query.isLoading, failure: classifyOutcomeFailure(query.error), refetch: () => void query.refetch() };
	}
	return { plan: query.data, isLoading: query.isLoading, refetch: () => void query.refetch() };
}

export interface ProposePlanState {
	pending: boolean;
	failure?: OutcomeFailure;
	reset: () => void;
	propose: (input: { expectedContractRevision: number }) => Promise<PlanRecord>;
}

/** Deterministic proposal for one contract revision; replays server-side. */
export function useProposeOutcomePlan(outcomeId: string | undefined): ProposePlanState {
	const queryClient = useQueryClient();
	const mutation = useMutation({
		mutationFn: async (input: { expectedContractRevision: number }) => {
			const { data, error } = await apiClient.POST("/api/v1/outcomes/{outcomeId}/plans", {
				params: { path: { outcomeId: outcomeId as string } },
				body: { expectedContractRevision: input.expectedContractRevision },
			});
			if (error) throw error;
			return (data as PlanEnvelope).plan;
		},
		onSuccess: (plan) => queryClient.setQueryData(planQueryKey(outcomeId), plan),
	});
	return {
		pending: mutation.isPending,
		failure: mutation.error ? classifyOutcomeFailure(mutation.error) : undefined,
		reset: () => mutation.reset(),
		propose: mutation.mutateAsync,
	};
}

export interface ApprovePlanState {
	pending: boolean;
	failure?: OutcomeFailure;
	reset: () => void;
	approve: (input: { planId: string; expectedContractRevision: number }) => Promise<PlanRecord>;
}

/** Owner approval against the revision the approver was looking at. */
export function useApproveOutcomePlan(outcomeId: string | undefined): ApprovePlanState {
	const queryClient = useQueryClient();
	const mutation = useMutation({
		mutationFn: async (input: { planId: string; expectedContractRevision: number }) => {
			const { data, error } = await apiClient.POST(
				"/api/v1/outcomes/{outcomeId}/plans/{planId}/approval",
				{
					params: { path: { outcomeId: outcomeId as string, planId: input.planId } },
					body: { expectedContractRevision: input.expectedContractRevision },
				},
			);
			if (error) throw error;
			return (data as PlanEnvelope).plan;
		},
		onSuccess: (plan) => queryClient.setQueryData(planQueryKey(outcomeId), plan),
	});
	return {
		pending: mutation.isPending,
		failure: mutation.error ? classifyOutcomeFailure(mutation.error) : undefined,
		reset: () => mutation.reset(),
		approve: mutation.mutateAsync,
	};
}
