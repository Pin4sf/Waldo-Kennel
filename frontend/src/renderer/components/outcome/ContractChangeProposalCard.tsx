import { Button } from "../ui/button";
import type { PlanningTurn } from "../../hooks/usePlanning";
import { useTranslation } from "react-i18next";

export function ContractChangeProposalCard({ turn, onReviewContract }: { turn: PlanningTurn; onReviewContract?: () => void }) {
	const { t } = useTranslation();
	if (!turn.contractChange) return null;
	return (
		<section className="rounded-md border border-warning/40 bg-warning/5 px-3 py-3" data-testid="planning-contract-change">
			<h4 className="text-sm font-medium">{t("planning.contractChangeTitle")}</h4>
			<p className="mt-1 text-sm">{turn.contractChange.summary}</p>
			{turn.contractChange.changedFields.length > 0 && <p className="mt-2 text-xs text-muted-foreground">{t("planning.changedFields")}: {turn.contractChange.changedFields.join(", ")}</p>}
			<p className="mt-2 text-xs text-muted-foreground">{t("planning.contractChangeNote")}</p>
			{onReviewContract && <Button className="mt-3" onClick={onReviewContract} size="sm" type="button" variant="outline">{t("planning.reviewContract")}</Button>}
		</section>
	);
}
