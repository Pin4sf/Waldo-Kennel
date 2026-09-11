import { CheckCircle2, FolderOpen, Loader2, RefreshCw } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import {
	useOutcomeDeliveries,
	useRequestOutcomeDelivery,
	type OutcomeDeliveryRecord,
	type RequestOutcomeDeliveryInput,
} from "../../hooks/useOutcomeArtifacts";
import { useOutcomeAttempts, useOutcomeProof } from "../../hooks/useOutcome";
import { useOutcomeRunState } from "../../hooks/useOutcomeRunState";
import { aoBridge } from "../../lib/bridge";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { Label } from "../ui/label";

function latestAcceptedDecision(proof: ReturnType<typeof useOutcomeProof>["proof"]) {
	return proof?.decisions
		.filter((decision) => decision.kind === "accept")
		.sort((left, right) => left.createdAt.localeCompare(right.createdAt))
		.at(-1);
}

function stateVariant(state: OutcomeDeliveryRecord["state"]) {
	if (state === "succeeded") return "success" as const;
	if (state === "failed") return "error" as const;
	if (state === "pending") return "accent" as const;
	return "outline" as const;
}

/**
 * Delivery is the final transfer boundary, not a synonym for acceptance. The
 * owner supplies the exact retained Attempt/artifact identity and destination;
 * the daemon owns validation, copy, recovery and the durable delivery ledger.
 */
export function OutcomeDeliveryPanel({ outcomeId }: { outcomeId: string }) {
	const { t } = useTranslation();
	const deliveriesQuery = useOutcomeDeliveries(outcomeId);
	const attemptsQuery = useOutcomeAttempts(outcomeId);
	const proofQuery = useOutcomeProof(outcomeId);
	const runQuery = useOutcomeRunState(outcomeId);
	const requestMutation = useRequestOutcomeDelivery(outcomeId);
	const [attemptId, setAttemptId] = useState("");
	const [artifactVersion, setArtifactVersion] = useState("");
	const [destination, setDestination] = useState("");
	const [disposition, setDisposition] = useState<"accepted" | "draft">("draft");

	const deliveries = deliveriesQuery.deliveries;
	const acceptanceDecision = latestAcceptedDecision(proofQuery.proof);
	const exportEligibility = runQuery.data?.eligibleActions.find((item) => item.action === "export");
	const latestAttempt = attemptsQuery.attempts?.find((attempt) => attempt.status === "succeeded") ?? attemptsQuery.attempts?.at(-1);

	useEffect(() => {
		if (attemptId) return;
		setAttemptId(deliveries[0]?.attemptId ?? latestAttempt?.id ?? "");
	}, [attemptId, deliveries, latestAttempt?.id]);
	useEffect(() => {
		if (artifactVersion) return;
		setArtifactVersion(deliveries[0]?.artifactVersion ?? "");
	}, [artifactVersion, deliveries]);
	useEffect(() => {
		if (!acceptanceDecision && disposition === "accepted") setDisposition("draft");
	}, [acceptanceDecision, disposition]);

	const pending = requestMutation.pending;
	const failure = deliveriesQuery.failure ?? attemptsQuery.failure ?? requestMutation.failure;
	const featureUnavailable = exportEligibility && !exportEligibility.available;
	const exactFieldsReady = Boolean(attemptId.trim() && artifactVersion.trim() && destination.trim());
	const canRequest = Boolean(exportEligibility?.available) && exactFieldsReady;
	const unavailableReason = exportEligibility?.reason
		? t(`mission.delivery.reason.${exportEligibility.reason}`, { defaultValue: exportEligibility.reason })
		: "unavailable";

	async function chooseDestination() {
		const path = await aoBridge.app.chooseDirectory(t("mission.delivery.chooseDestination"));
		if (path) setDestination(path);
	}

	async function request() {
		if (!canRequest || pending) return;
		const input: RequestOutcomeDeliveryInput = {
			attemptId: attemptId.trim(),
			artifactVersion: artifactVersion.trim(),
			destination: destination.trim(),
			disposition,
			...(disposition === "accepted" && acceptanceDecision
				? { acceptanceDecisionId: acceptanceDecision.id }
				: {}),
		};
		try {
			await requestMutation.request(input);
		} catch {
			// The typed daemon refusal is rendered below and remains retryable.
		}
	}

	function reuseDelivery(delivery: OutcomeDeliveryRecord) {
		setAttemptId(delivery.attemptId);
		setArtifactVersion(delivery.artifactVersion);
		setDestination(delivery.destination);
		setDisposition(acceptanceDecision && delivery.disposition === "accepted" ? "accepted" : "draft");
	}

	return (
		<section className="mt-5 rounded-md border border-border p-4" data-testid="outcome-delivery-panel">
			<div className="flex flex-wrap items-start justify-between gap-3">
				<div>
					<div className="flex items-center gap-2">
						<CheckCircle2 aria-hidden="true" className="size-4 text-muted-foreground" />
						<h3 className="text-sm font-medium">{t("mission.delivery.heading")}</h3>
					</div>
					<p className="mt-1 text-xs text-muted-foreground">{t("mission.delivery.intro")}</p>
				</div>
				{deliveriesQuery.failure && (
					<Button onClick={deliveriesQuery.refetch} size="sm" variant="ghost">
						<RefreshCw aria-hidden="true" className="size-3.5" />
						{t("mission.refresh")}
					</Button>
				)}
			</div>

			<div className="mt-3 rounded border border-border bg-muted/20 p-3">
				<p className="text-xs text-muted-foreground">{t("mission.delivery.exactArtifact")}</p>
				<div className="mt-3 grid gap-3 sm:grid-cols-2">
					<div>
						<Label htmlFor="delivery-attempt-id">{t("mission.delivery.attemptId")}</Label>
						<Input id="delivery-attempt-id" onChange={(event) => setAttemptId(event.target.value)} value={attemptId} />
					</div>
					<div>
						<Label htmlFor="delivery-artifact-version">{t("mission.delivery.artifactVersion")}</Label>
						<Input id="delivery-artifact-version" onChange={(event) => setArtifactVersion(event.target.value)} value={artifactVersion} />
					</div>
				</div>
				<div className="mt-3">
					<Label htmlFor="delivery-destination">{t("mission.delivery.destination")}</Label>
					<div className="mt-1 flex gap-2">
						<Input id="delivery-destination" onChange={(event) => setDestination(event.target.value)} value={destination} />
						<Button aria-label={t("mission.delivery.chooseDestination")} onClick={() => void chooseDestination()} size="icon" type="button" variant="outline">
							<FolderOpen aria-hidden="true" className="size-4" />
						</Button>
					</div>
				</div>
				<fieldset className="mt-3" disabled={pending}>
					<legend className="text-xs font-medium">{t("mission.delivery.disposition")}</legend>
					<div className="mt-2 flex flex-wrap gap-3 text-xs">
						<label className="flex items-center gap-2">
							<input checked={disposition === "draft"} name="delivery-disposition" onChange={() => setDisposition("draft")} type="radio" />
							{t("mission.delivery.draft")}
						</label>
						<label className="flex items-center gap-2">
							<input checked={disposition === "accepted"} disabled={!acceptanceDecision} name="delivery-disposition" onChange={() => setDisposition("accepted")} type="radio" />
							{t("mission.delivery.accepted")}
						</label>
					</div>
					<p className="mt-1 text-2xs text-muted-foreground">
						{acceptanceDecision ? t("mission.delivery.acceptedAvailable") : t("mission.delivery.acceptedUnavailable")}
					</p>
				</fieldset>
				<div className="mt-3 flex flex-wrap items-center gap-2">
					<Button data-testid="outcome-delivery-request" disabled={!canRequest || pending} onClick={() => void request()} size="sm">
						{pending && <Loader2 aria-hidden="true" className="size-3.5 animate-spin" />}
						{t("mission.delivery.request")}
					</Button>
					{featureUnavailable && <span className="text-2xs text-warning">{t("mission.delivery.unavailableReason", { reason: unavailableReason })}</span>}
					{!exportEligibility && <span className="text-2xs text-muted-foreground">{t("mission.delivery.waitingForEligibility")}</span>}
				</div>
			</div>

			<p className="mt-2 text-2xs text-muted-foreground">{t("mission.delivery.boundary")}</p>
			{deliveriesQuery.isLoading && <p className="mt-3 text-xs text-muted-foreground">{t("mission.delivery.loading")}</p>}
			{failure && <p className="mt-3 text-xs text-destructive" role="alert">{failure.message}</p>}

			<div className="mt-4">
				<h4 className="text-xs font-medium">{t("mission.delivery.history")}</h4>
				{deliveries.length === 0 ? (
					<p className="mt-2 text-xs text-muted-foreground">{t("mission.delivery.empty")}</p>
				) : (
					<ul className="mt-2 space-y-2">
						{deliveries.map((delivery) => <DeliveryHistoryRow delivery={delivery} key={delivery.id} onReuse={reuseDelivery} />)}
					</ul>
				)}
			</div>
		</section>
	);
}

function DeliveryHistoryRow({ delivery, onReuse }: { delivery: OutcomeDeliveryRecord; onReuse: (delivery: OutcomeDeliveryRecord) => void }) {
	const { t } = useTranslation();
	const sameDestinationRetry = delivery.state === "failed" && delivery.failureCode === "DELIVERY_INTERRUPTED";
	const recoveryNote = delivery.completionSource === "recovered"
		? t("mission.delivery.recovered")
		: delivery.completionSource === "observed"
			? t("mission.delivery.observed")
			: undefined;

	return (
		<li className="rounded border border-border p-3 text-xs" data-testid={`delivery-${delivery.id}`}>
			<div className="flex flex-wrap items-center justify-between gap-2">
				<div className="flex flex-wrap items-center gap-2">
					<Badge variant={stateVariant(delivery.state)}>{delivery.state}</Badge>
					<Badge variant="outline">{delivery.disposition}</Badge>
					<span className="text-muted-foreground">{new Date(delivery.requestedAt).toLocaleString()}</span>
				</div>
				{sameDestinationRetry && <Button onClick={() => onReuse(delivery)} size="sm" variant="outline">{t("mission.delivery.retrySame")}</Button>}
			</div>
			<p className="mt-2 break-all">{delivery.destination}</p>
			<p className="mt-1 text-2xs text-muted-foreground">{t("mission.delivery.artifactSummary", { attempt: delivery.attemptId, version: delivery.artifactVersion })}</p>
			{recoveryNote && <p className="mt-1 text-2xs text-muted-foreground">{recoveryNote}</p>}
			{delivery.failureCode && <p className="mt-2 text-warning">{delivery.failureCode}: {delivery.failureDetail ?? t("mission.delivery.noFailureDetail")}</p>}
			{delivery.failureCode === "DELIVERY_RECOVERY_MISMATCH" && <p className="mt-1 text-2xs text-muted-foreground">{t("mission.delivery.chooseDifferentDestination")}</p>}
			{delivery.failureCode === "DELIVERY_RECOVERY_UNREADABLE" && <p className="mt-1 text-2xs text-muted-foreground">{t("mission.delivery.unreadable")}</p>}
			{delivery.failureCode === "DELIVERY_RECOVERY_UNVERIFIABLE" && <p className="mt-1 text-2xs text-muted-foreground">{t("mission.delivery.reproduce")}</p>}
			{delivery.state === "pending" && <p className="mt-1 text-2xs text-muted-foreground">{t("mission.delivery.pendingNoRetry")}</p>}
		</li>
	);
}
