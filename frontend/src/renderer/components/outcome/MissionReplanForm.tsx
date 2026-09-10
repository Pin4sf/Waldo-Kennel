import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { apiClient } from "../../lib/api-client";
import { classifyOutcomeFailure, outcomeQueryKey, planQueryKey } from "../../hooks/useOutcome";
import { Button } from "../ui/button";

export function MissionReplanForm({ outcomeId, revision }: { outcomeId: string; revision: number }) {
	const { t } = useTranslation();
	const cache = useQueryClient();
	const [feedback, setFeedback] = useState("");
	const mutation = useMutation({
		retry: false,
		mutationFn: async () => {
			const { data, error } = await apiClient.POST("/api/v1/outcomes/{outcomeId}/plans/replan", {
				params: { path: { outcomeId } },
				body: { expectedContractRevision: revision, feedback },
			});
			if (error) throw error;
			return data;
		},
		onSuccess: () => {
			// Refetch current facts instead of installing a possibly delayed response.
			void cache.invalidateQueries({ queryKey: planQueryKey(outcomeId) });
			void cache.invalidateQueries({ queryKey: outcomeQueryKey(outcomeId) });
			setFeedback("");
		},
	});
	return (
		<form
			className="mt-4 border-t border-border pt-3"
			onSubmit={(event) => {
				event.preventDefault();
				if (feedback.trim() && !mutation.isPending && !mutation.error) mutation.mutate();
			}}
		>
			<label className="block text-sm font-medium">
				{t("mission.replanFeedback")}
				<textarea
					className="mt-2 block min-h-20 w-full rounded-md border border-border bg-card p-2 text-sm focus-visible:ring-2 focus-visible:ring-ring"
					value={feedback}
					disabled={mutation.isPending}
					onChange={(event) => setFeedback(event.target.value)}
				/>
			</label>
			<p className="my-2 text-xs text-muted-foreground">{t("mission.replanNote", { revision })}</p>
			<Button size="sm" disabled={!feedback.trim() || mutation.isPending || Boolean(mutation.error)} type="submit">
				{t(mutation.isPending ? "mission.replanning" : "mission.replan")}
			</Button>
			{mutation.error && (
				<div role="alert" className="mt-2 text-xs text-destructive">
					<p>{classifyOutcomeFailure(mutation.error).message}</p>
					<p>{t("mission.replanReconcile")}</p>
					<Button
						size="sm"
						variant="outline"
						onClick={() => {
							void cache.invalidateQueries({ queryKey: planQueryKey(outcomeId) });
							void cache.invalidateQueries({ queryKey: outcomeQueryKey(outcomeId) });
						}}
					>
						{t("mission.refresh")}
					</Button>
				</div>
			)}
		</form>
	);
}
