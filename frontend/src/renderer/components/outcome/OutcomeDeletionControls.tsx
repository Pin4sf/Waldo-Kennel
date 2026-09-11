import { useTranslation } from "react-i18next";
import { useState } from "react";
import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query";
import { apiClient, apiErrorMessage } from "../../lib/api-client";
import { Button } from "../ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription,
} from "../ui/dialog";

export function OutcomeDeletionControls({
  outcomeId,
  onRemoved,
}: {
  outcomeId: string;
  onRemoved: () => void;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button variant="ghost" size="sm" onClick={() => setOpen(true)}>
        {t("deletion.open")}
      </Button>
      {open && (
        <DeletionDialog
          outcomeId={outcomeId}
          onClose={() => setOpen(false)}
          onChanged={() => {
            setOpen(false);
            onRemoved();
          }}
        />
      )}
    </>
  );
}
function DeletionDialog({
  outcomeId,
  onClose,
  onChanged,
}: {
  outcomeId: string;
  onClose: () => void;
  onChanged: () => void;
}) {
  const { t } = useTranslation();
  const [confirmation, setConfirmation] = useState("");
  const client = useQueryClient();
  const query = useQuery({
    queryKey: ["outcome-deletion", outcomeId],
    retry: false,
    queryFn: async () => {
      const { data, error } = await apiClient.GET(
        "/api/v1/outcomes/{outcomeId}/deletion",
        { params: { path: { outcomeId } } },
      );
      if (error || !data)
        throw new Error(apiErrorMessage(error) ?? t("deletion.loadFailed"));
      return data.deletion;
    },
  });
  const mutation = useMutation({
    mutationFn: async (action: "trash" | "restore" | "permanent") => {
      if (!query.data) throw new Error(t("deletion.previewRequired"));
      const { error } = await apiClient.POST(
        "/api/v1/outcomes/{outcomeId}/deletion",
        {
          params: { path: { outcomeId } },
          body: { action, revision: query.data.revision, confirmation },
        },
      );
      if (error)
        throw new Error(apiErrorMessage(error) ?? t("deletion.failed"));
    },
    onSuccess: () => {
      void client.invalidateQueries({
        predicate: (q) =>
          [
            "project-outcomes",
            "outcome-trash",
            "outcome-deletion",
            "outcome",
          ].includes(String(q.queryKey[0])),
      });
      onChanged();
    },
    onError: () => {
      void query.refetch();
    },
  });
  const p = query.data;
  const blocked = Boolean(p?.blockers?.length);
  return (
    <Dialog
      open
      onOpenChange={(next) => {
        if (!next && !mutation.isPending) onClose();
      }}
    >
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-xl">
        <DialogTitle>
          {p?.erasing ? t("deletion.finish") : t("deletion.title")}
        </DialogTitle>
        <DialogDescription>{t("deletion.description")}</DialogDescription>
        {query.isLoading && <p>{t("deletion.checking")}</p>}
        {query.error && <p role="alert">{query.error.message}</p>}
        {p && (
          <>
            <div className="rounded-lg border border-border bg-muted/40 px-4 py-3">
              <p className="text-xs text-muted-foreground">
                {t("deletion.outcomeLabel")}
              </p>
              <p className="mt-1 text-base font-medium break-words">
                {p.title}
              </p>
            </div>
            <p className="text-sm text-muted-foreground">
              {t("deletion.preserved")}
            </p>
            <details className="text-sm text-muted-foreground">
              <summary className="cursor-pointer">
                {t("deletion.details")}
              </summary>
              <p className="mt-2">
                {t("deletion.counts", {
                  outcomes: p.outcomeCount,
                  records: p.recordCount,
                  sessions: p.sessionIds?.length ?? 0,
                })}
              </p>
              <p className="mt-2">{t("deletion.retainedDetails")}</p>
              {!!p.workspacePaths?.length && (
                <ul className="mt-2 text-xs break-all">
                  {p.workspacePaths.map((path) => (
                    <li key={path}>{path}</li>
                  ))}
                </ul>
              )}
            </details>
            {blocked && (
              <div
                role="alert"
                className="rounded-md border border-border p-3 text-sm"
              >
                <p className="font-medium">{t("deletion.blockers")}</p>
                <ul>
                  {p.blockers.map((reason, index) => (
                    <li key={index}>{reason}</li>
                  ))}
                </ul>
              </div>
            )}
            {p.erasing && <p role="status">{t("deletion.pending")}</p>}
            {!p.erasing && (
              <div className="flex gap-2">
                <Button
                  disabled={mutation.isPending || (!p.trashed && blocked)}
                  onClick={() =>
                    mutation.mutate(p.trashed ? "restore" : "trash")
                  }
                >
                  {p.trashed ? t("deletion.restore") : t("deletion.trash")}
                </Button>
              </div>
            )}
            <div className="space-y-2 border-t border-border pt-4">
              <p className="text-sm">{t("deletion.warning")}</p>
              <label className="block text-sm">
                {t("deletion.typeTitle")}
                <input
                  aria-label={t("deletion.confirmTitle")}
                  className="mt-2 w-full rounded-md border border-border bg-background px-3 py-2"
                  value={confirmation}
                  onChange={(event) => setConfirmation(event.target.value)}
                  disabled={mutation.isPending}
                />
              </label>
              <Button
                variant="primary"
                className="bg-destructive text-white"
                disabled={
                  mutation.isPending ||
                  blocked ||
                  confirmation.trim().toLowerCase() !== "confirm"
                }
                onClick={() => mutation.mutate("permanent")}
              >
                {mutation.isPending
                  ? t("deletion.working")
                  : p.erasing
                    ? t("deletion.retry")
                    : t("deletion.permanent")}
              </Button>
            </div>
          </>
        )}
        {mutation.error && (
          <p role="alert" className="text-sm text-destructive">
            {mutation.error.message}
          </p>
        )}
      </DialogContent>
    </Dialog>
  );
}
export function OutcomeTrash({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [selected, setSelected] = useState<string>();
  const query = useQuery({
    queryKey: ["outcome-trash", projectId],
    enabled: open,
    retry: false,
    queryFn: async () => {
      const { data, error } = await apiClient.GET(
        "/api/v1/projects/{id}/outcome-trash",
        { params: { path: { id: projectId } } },
      );
      if (error || !data)
        throw new Error(
          apiErrorMessage(error) ?? t("deletion.trashLoadFailed"),
        );
      return data.outcomes;
    },
  });
  return (
    <>
      <Button size="sm" variant="ghost" onClick={() => setOpen(true)}>
        {t("deletion.trashLink")}
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-h-[85vh] overflow-y-auto">
          <DialogTitle>{t("deletion.trashTitle")}</DialogTitle>
          <DialogDescription>
            {t("deletion.trashDescription")}
          </DialogDescription>
          {query.isLoading && <p>{t("deletion.loadingTrash")}</p>}
          {query.error && <p role="alert">{query.error.message}</p>}
          {query.data?.length === 0 && <p>{t("deletion.emptyTrash")}</p>}
          {query.data?.map((item) => (
            <div
              key={item.outcomeId}
              className="flex items-center justify-between gap-3 border-b border-border py-3"
            >
              <span className="min-w-0 break-words">
                {item.title}
                {item.erasing ? t("deletion.cleanupPending") : ""}
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setSelected(item.outcomeId)}
              >
                {t("deletion.manage")}
              </Button>
            </div>
          ))}
        </DialogContent>
      </Dialog>
      {selected && (
        <DeletionDialog
          outcomeId={selected}
          onClose={() => setSelected(undefined)}
          onChanged={() => {
            setSelected(undefined);
            void query.refetch();
          }}
        />
      )}
    </>
  );
}
