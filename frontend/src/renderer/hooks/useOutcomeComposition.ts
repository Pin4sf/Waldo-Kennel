/**
 * Composed Outcome reads and writes (ADR 0007).
 *
 * Like every other Outcome hook here, the renderer is a thin surface over the
 * daemon: shape, attention, coverage, gates, and batch eligibility all arrive
 * DERIVED. Nothing in this file recomputes readiness, decides who may be
 * accepted, or claims a decision the daemon has not answered.
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { components } from "../../api/schema";
import { apiClient, apiErrorCode } from "../lib/api-client";
import { classifyOutcomeFailure, outcomeProofQueryKey, outcomeQueryKey, type OutcomeFailure } from "./useOutcome";

export type OutcomeCompositionRecord = components["schemas"]["OutcomeCompositionResponse"];
export type ContributorRecord = components["schemas"]["ContributorResponse"];
export type ContributorAttentionRecord = components["schemas"]["ContributorAttentionResponse"];
export type CriterionClaimRecord = components["schemas"]["CriterionClaimResponse"];
export type DecompositionRecord = components["schemas"]["DecompositionResponse"];
export type BatchEntryVerdictRecord = components["schemas"]["BatchEntryVerdictResponse"];
export type AcceptBatchRecord = components["schemas"]["AcceptBatchResponse"];
export type ProposeDecompositionRequest = components["schemas"]["ProposeDecompositionRequest"];
export type WaiveDependencyRequest = components["schemas"]["WaiveContributionDependencyRequest"];
export type AcceptBatchRequest = components["schemas"]["AcceptContributorBatchRequest"];

type CompositionEnvelope = components["schemas"]["OutcomeCompositionEnvelope"];
type DecompositionEnvelope = components["schemas"]["DecompositionEnvelope"];
type BatchEligibilityEnvelope = components["schemas"]["BatchEligibilityEnvelope"];
type AcceptBatchEnvelope = components["schemas"]["AcceptBatchEnvelope"];

/**
 * The daemon's derived shape vocabulary, mirrored so controls never key off
 * scattered literals. Source of truth: OutcomeCompositionResponse.shape.
 */
export const OUTCOME_SHAPES = { direct: "direct", decomposed: "decomposed" } as const;

/** The daemon's attention vocabulary, ordered most demanding first. */
export const ATTENTION_KINDS = {
	needsYou: "needs_you",
	actionRequired: "action_required",
	readyForAcceptance: "ready_for_acceptance",
	waiting: "waiting",
	running: "running",
	accepted: "accepted",
} as const;

/** The daemon's decomposition statuses. */
export const DECOMPOSITION_STATUS = { proposed: "proposed", authorized: "authorized" } as const;

export function outcomeCompositionQueryKey(outcomeId: string | undefined) {
	return ["outcome-composition", outcomeId ?? ""] as const;
}
export function outcomeDecompositionQueryKey(outcomeId: string | undefined) {
	return ["outcome-decomposition", outcomeId ?? ""] as const;
}
export function outcomeBatchEligibilityQueryKey(outcomeId: string | undefined) {
	return ["outcome-batch-eligibility", outcomeId ?? ""] as const;
}

/** Codes a retry cannot help. */
const PERMANENT = new Set([
	"OUTCOME_NOT_FOUND",
	"DECOMPOSITION_NOT_FOUND",
	"OUTCOME_NOT_DECOMPOSED",
	"OUTCOME_PROOF_UNAVAILABLE",
]);

function retryUnlessPermanent(attempt: number, error: unknown) {
	const code = apiErrorCode(error);
	if (code && PERMANENT.has(code)) return false;
	return attempt < 2;
}

export interface OutcomeCompositionResult {
	composition?: OutcomeCompositionRecord;
	isLoading: boolean;
	failure?: OutcomeFailure;
	refetch: () => void;
}

/** Derived shape, contributors, coverage, gates, and attention roll-up. */
export function useOutcomeComposition(outcomeId: string | undefined): OutcomeCompositionResult {
	const query = useQuery({
		queryKey: outcomeCompositionQueryKey(outcomeId),
		enabled: Boolean(outcomeId),
		queryFn: async () => {
			const { data, error } = await apiClient.GET("/api/v1/outcomes/{outcomeId}/composition", {
				params: { path: { outcomeId: outcomeId as string } },
			});
			if (error) throw error;
			return (data as CompositionEnvelope).composition;
		},
		retry: retryUnlessPermanent,
	});
	return {
		composition: query.data,
		isLoading: query.isLoading,
		failure: query.error ? classifyOutcomeFailure(query.error) : undefined,
		refetch: () => void query.refetch(),
	};
}

export interface OutcomeDecompositionResult {
	decomposition?: DecompositionRecord;
	isLoading: boolean;
	failure?: OutcomeFailure;
	refetch: () => void;
}

/**
 * The newest decomposition of any status. A 404 is the ordinary answer for an
 * Outcome nobody has decomposed, not a failure worth retrying.
 */
export function useOutcomeDecomposition(outcomeId: string | undefined): OutcomeDecompositionResult {
	const query = useQuery({
		queryKey: outcomeDecompositionQueryKey(outcomeId),
		enabled: Boolean(outcomeId),
		queryFn: async () => {
			const { data, error } = await apiClient.GET("/api/v1/outcomes/{outcomeId}/decomposition", {
				params: { path: { outcomeId: outcomeId as string } },
			});
			if (error) throw error;
			return (data as DecompositionEnvelope).decomposition;
		},
		retry: retryUnlessPermanent,
	});
	return {
		decomposition: query.data,
		isLoading: query.isLoading,
		failure: query.error ? classifyOutcomeFailure(query.error) : undefined,
		refetch: () => void query.refetch(),
	};
}

/** Who could be accepted together right now, and why the others could not. */
export function useBatchEligibility(outcomeId: string | undefined, enabled = true) {
	const query = useQuery({
		queryKey: outcomeBatchEligibilityQueryKey(outcomeId),
		enabled: Boolean(outcomeId) && enabled,
		queryFn: async () => {
			const { data, error } = await apiClient.GET("/api/v1/outcomes/{outcomeId}/acceptance-batch", {
				params: { path: { outcomeId: outcomeId as string } },
			});
			if (error) throw error;
			return (data as BatchEligibilityEnvelope).contributors;
		},
		retry: retryUnlessPermanent,
	});
	return {
		contributors: query.data ?? [],
		isLoading: query.isLoading,
		failure: query.error ? classifyOutcomeFailure(query.error) : undefined,
		refetch: () => void query.refetch(),
	};
}

/**
 * Invalidate every projection a composition write can move. Composition,
 * decomposition, the parent's proof, and its own record are all derived from
 * the same durable facts, so one write can change all four.
 */
function useCompositionInvalidation(outcomeId: string | undefined) {
	const queryClient = useQueryClient();
	return () => {
		void queryClient.invalidateQueries({ queryKey: outcomeCompositionQueryKey(outcomeId) });
		void queryClient.invalidateQueries({ queryKey: outcomeDecompositionQueryKey(outcomeId) });
		void queryClient.invalidateQueries({ queryKey: outcomeBatchEligibilityQueryKey(outcomeId) });
		void queryClient.invalidateQueries({ queryKey: outcomeProofQueryKey(outcomeId) });
		void queryClient.invalidateQueries({ queryKey: outcomeQueryKey(outcomeId) });
	};
}

export interface DecompositionMutationState<TInput, TResult> {
	pending: boolean;
	failure?: OutcomeFailure;
	reset: () => void;
	submit: (input: TInput) => Promise<TResult>;
}

/** Propose a decomposition. It creates nothing until authorized. */
export function useProposeDecomposition(
	outcomeId: string | undefined,
): DecompositionMutationState<ProposeDecompositionRequest, DecompositionRecord> {
	const invalidate = useCompositionInvalidation(outcomeId);
	const mutation = useMutation({
		mutationFn: async (input: ProposeDecompositionRequest) => {
			const { data, error } = await apiClient.POST("/api/v1/outcomes/{outcomeId}/decompositions", {
				params: { path: { outcomeId: outcomeId as string } },
				body: input,
			});
			if (error) throw error;
			return (data as DecompositionEnvelope).decomposition;
		},
		onSuccess: invalidate,
	});
	return {
		pending: mutation.isPending,
		failure: mutation.error ? classifyOutcomeFailure(mutation.error) : undefined,
		reset: () => mutation.reset(),
		submit: mutation.mutateAsync,
	};
}

/** Authorize a decomposition — the decision that creates the contributors. */
export function useAuthorizeDecomposition(
	outcomeId: string | undefined,
): DecompositionMutationState<string, DecompositionRecord> {
	const invalidate = useCompositionInvalidation(outcomeId);
	const mutation = useMutation({
		mutationFn: async (decompositionId: string) => {
			const { data, error } = await apiClient.POST(
				"/api/v1/outcomes/{outcomeId}/decompositions/{decompositionId}/authorization",
				{ params: { path: { outcomeId: outcomeId as string, decompositionId } } },
			);
			if (error) throw error;
			return (data as DecompositionEnvelope).decomposition;
		},
		onSuccess: invalidate,
	});
	return {
		pending: mutation.isPending,
		failure: mutation.error ? classifyOutcomeFailure(mutation.error) : undefined,
		reset: () => mutation.reset(),
		submit: mutation.mutateAsync,
	};
}

/** Waive one declared ordering. The reason is durable and required. */
export function useWaiveContributionDependency(
	outcomeId: string | undefined,
): DecompositionMutationState<WaiveDependencyRequest, DecompositionRecord> {
	const invalidate = useCompositionInvalidation(outcomeId);
	const mutation = useMutation({
		mutationFn: async (input: WaiveDependencyRequest) => {
			const { data, error } = await apiClient.POST("/api/v1/outcomes/{outcomeId}/decomposition/waivers", {
				params: { path: { outcomeId: outcomeId as string } },
				body: input,
			});
			if (error) throw error;
			return (data as DecompositionEnvelope).decomposition;
		},
		onSuccess: invalidate,
	});
	return {
		pending: mutation.isPending,
		failure: mutation.error ? classifyOutcomeFailure(mutation.error) : undefined,
		reset: () => mutation.reset(),
		submit: mutation.mutateAsync,
	};
}

/**
 * One owner sitting. The daemon writes one decision per Outcome accepted and
 * reports every contributor it withheld; this hook never decides anything.
 */
export function useAcceptContributorBatch(
	outcomeId: string | undefined,
): DecompositionMutationState<AcceptBatchRequest, AcceptBatchRecord> {
	const invalidate = useCompositionInvalidation(outcomeId);
	const mutation = useMutation({
		mutationFn: async (input: AcceptBatchRequest) => {
			const { data, error } = await apiClient.POST("/api/v1/outcomes/{outcomeId}/acceptance-batch", {
				params: { path: { outcomeId: outcomeId as string } },
				body: input,
			});
			if (error) throw error;
			return (data as AcceptBatchEnvelope).batch;
		},
		onSuccess: invalidate,
	});
	return {
		pending: mutation.isPending,
		failure: mutation.error ? classifyOutcomeFailure(mutation.error) : undefined,
		reset: () => mutation.reset(),
		submit: mutation.mutateAsync,
	};
}
