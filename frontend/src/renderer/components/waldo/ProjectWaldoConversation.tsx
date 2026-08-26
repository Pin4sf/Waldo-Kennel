import { AlertTriangle, CheckCircle2, Link2, Link2Off, Send } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { components } from "../../../api/schema";
import {
  apiClient,
  apiErrorCode,
  apiErrorMessage,
  apiErrorRequestId,
} from "../../lib/api-client";
import { cn } from "../../lib/utils";

type Snapshot = components["schemas"]["WaldoConversationSnapshotResponse"];
type ContextAttachment = components["schemas"]["WaldoContextAttachmentResponse"];

type ProjectWaldoConversationProps = {
  daemonReady: boolean;
  onOpenHome?: () => void;
  outcomeId?: string;
  outcomeTitle?: string;
  projectId: string;
  projectName: string;
};

function snapshotKey(projectId: string) {
  return `kennel.waldo.snapshot.${projectId}`;
}

function draftKey(projectId: string, outcomeId?: string) {
  return `kennel.waldo.draft.${projectId}.${outcomeId ?? "project"}`;
}

function requestKey(prefix: string) {
  const id = globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  return `${prefix}-${id}`;
}

function readSnapshot(projectId: string): Snapshot | null {
  try {
    const raw = localStorage.getItem(snapshotKey(projectId));
    return raw ? (JSON.parse(raw) as Snapshot) : null;
  } catch {
    return null;
  }
}

function writeSnapshot(projectId: string, snapshot: Snapshot) {
  try {
    localStorage.setItem(snapshotKey(projectId), JSON.stringify(snapshot));
  } catch {
    // SQLite remains canonical. A failed renderer cache only removes offline read-back.
  }
}

export function ProjectWaldoConversation({ daemonReady, onOpenHome, outcomeId, outcomeTitle, projectId, projectName }: ProjectWaldoConversationProps) {
  const { t } = useTranslation();
  const [snapshot, setSnapshot] = useState<Snapshot | null>(() => readSnapshot(projectId));
  const [draft, setDraft] = useState(() => localStorage.getItem(draftKey(projectId, outcomeId)) ?? "");
  const [loading, setLoading] = useState(daemonReady);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [requestId, setRequestId] = useState("");
  const [notice, setNotice] = useState("");

  const acceptSnapshot = useCallback((next: Snapshot) => {
    setSnapshot(next);
    writeSnapshot(projectId, next);
  }, [projectId]);

  const load = useCallback(async () => {
    if (!daemonReady) {
      setLoading(false);
      return;
    }
    setLoading(true);
    setError("");
    setRequestId("");
    const read = await apiClient.GET("/api/v1/projects/{id}/waldo-conversation", {
      params: { path: { id: projectId } },
    });
    if (read.data) {
      acceptSnapshot(read.data.waldoConversation);
      setLoading(false);
      return;
    }
    if (apiErrorCode(read.error) !== "WALDO_CONVERSATION_NOT_FOUND") {
      setError(apiErrorMessage(read.error, t("waldo.project.error.read")));
      setRequestId(apiErrorRequestId(read.error) ?? "");
      setLoading(false);
      return;
    }
    const opened = await apiClient.POST("/api/v1/projects/{id}/waldo-conversation", {
      params: { path: { id: projectId } },
    });
    if (opened.data) acceptSnapshot(opened.data.waldoConversation);
    else {
      setError(apiErrorMessage(opened.error, t("waldo.project.error.open")));
      setRequestId(apiErrorRequestId(opened.error) ?? "");
    }
    setLoading(false);
  }, [acceptSnapshot, daemonReady, projectId, t]);

  useEffect(() => {
    setSnapshot(readSnapshot(projectId));
    setDraft(localStorage.getItem(draftKey(projectId, outcomeId)) ?? "");
    setNotice("");
    void load();
  }, [load, outcomeId, projectId]);

  useEffect(() => {
    try {
      localStorage.setItem(draftKey(projectId, outcomeId), draft);
    } catch {
      // Draft persistence is best-effort renderer state, never canonical truth.
    }
  }, [draft, outcomeId, projectId]);

  const activeEpisode = useMemo(
    () => [...(snapshot?.episodes ?? [])].reverse().find((episode) => episode.state === "active"),
    [snapshot?.episodes],
  );
  const activeAttachments = useMemo(
    () => (snapshot?.contextAttachments ?? []).filter((attachment) => attachment.active),
    [snapshot?.contextAttachments],
  );
  const selectedAttachment = activeAttachments.find(
    (attachment) => attachment.ref.kind === (outcomeId ? "outcome" : "project") && attachment.ref.objectId === (outcomeId ?? projectId),
  );
  const selectedContextLabel = outcomeId ? `Outcome · ${outcomeTitle || outcomeId}` : null;

  async function ensureActiveEpisode(current: Snapshot): Promise<{ episodeId: string; snapshot: Snapshot } | null> {
    const existing = [...current.episodes].reverse().find((episode) => episode.state === "active");
    if (existing) return { episodeId: existing.id, snapshot: current };
    const opened = await apiClient.POST("/api/v1/projects/{id}/waldo-conversation/episodes", {
      params: { path: { id: projectId } },
      body: { expectedRevision: current.conversation.revision, requestKey: requestKey("waldo-episode") },
    });
    if (!opened.data) {
      setError(apiErrorMessage(opened.error, t("waldo.project.error.episode")));
      setRequestId(apiErrorRequestId(opened.error) ?? "");
      return null;
    }
    acceptSnapshot(opened.data.waldoConversation);
    const episode = [...opened.data.waldoConversation.episodes].reverse().find((item) => item.state === "active");
    return episode ? { episodeId: episode.id, snapshot: opened.data.waldoConversation } : null;
  }

  async function sendTurn() {
    const message = draft.trim();
    if (!daemonReady || !snapshot || !message || busy) return;
    setBusy(true);
    setError("");
    setNotice("");
    const episode = await ensureActiveEpisode(snapshot);
    if (!episode) {
      setBusy(false);
      return;
    }
    const response = await apiClient.POST("/api/v1/projects/{id}/waldo-conversation/turns", {
      params: { path: { id: projectId } },
      body: {
        expectedRevision: episode.snapshot.conversation.revision,
        episodeId: episode.episodeId,
        role: "user",
        message,
        contextAttachmentIds: episode.snapshot.contextAttachments.filter((attachment) => attachment.active).map((attachment) => attachment.id),
        requestKey: requestKey("waldo-turn"),
      },
    });
    if (response.data) {
      acceptSnapshot(response.data.waldoConversation);
      setDraft("");
      setNotice(t("waldo.project.saved"));
    } else {
      setError(apiErrorMessage(response.error, t("waldo.project.error.save")));
      setRequestId(apiErrorRequestId(response.error) ?? "");
      if (apiErrorCode(response.error) === "WALDO_CONVERSATION_REVISION_CONFLICT") void load();
    }
    setBusy(false);
  }

  async function attachSelectedContext() {
    if (!daemonReady || !snapshot || busy) return;
    setBusy(true);
    setError("");
    const response = await apiClient.POST("/api/v1/projects/{id}/waldo-conversation/context", {
      params: { path: { id: projectId } },
      body: {
        expectedRevision: snapshot.conversation.revision,
        ref: { kind: outcomeId ? "outcome" : "project", objectId: outcomeId ?? projectId, provenance: { kind: "user", sourceId: "waldo-rail" } },
        requestKey: requestKey("waldo-context"),
      },
    });
    if (response.data) acceptSnapshot(response.data.waldoConversation);
    else {
      setError(apiErrorMessage(response.error, t("waldo.project.error.attach")));
      setRequestId(apiErrorRequestId(response.error) ?? "");
    }
    setBusy(false);
  }

  async function detachContext(attachment: ContextAttachment) {
    if (!daemonReady || !snapshot || busy) return;
    setBusy(true);
    setError("");
    const response = await apiClient.POST("/api/v1/projects/{id}/waldo-conversation/context/{attachmentId}/detach", {
      params: { path: { id: projectId, attachmentId: attachment.id } },
      body: {
        expectedRevision: snapshot.conversation.revision,
        reason: "Detached from Waldo rail",
        requestKey: requestKey("waldo-context-detach"),
      },
    });
    if (response.data) acceptSnapshot(response.data.waldoConversation);
    else {
      setError(apiErrorMessage(response.error, t("waldo.project.error.detach")));
      setRequestId(apiErrorRequestId(response.error) ?? "");
    }
    setBusy(false);
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="border-b border-border px-4 py-3">
        <div className="flex flex-wrap items-center gap-2">
          <span className="rounded-full bg-foreground px-2.5 py-1 text-xs font-semibold text-background">{projectName}</span>
          {selectedContextLabel ? <span className="rounded-full border border-border px-2.5 py-1 text-xs text-foreground">{selectedContextLabel}</span> : null}
          {selectedAttachment ? (
            <>
              <span className="inline-flex items-center gap-1 rounded-full border border-border px-2.5 py-1 text-xs text-foreground">
                <Link2 aria-hidden="true" className="size-3" /> {outcomeId ? t("waldo.project.outcomeContextAttached") : t("waldo.project.contextAttached")}
              </span>
              <button aria-label={outcomeId ? t("waldo.project.detachOutcomeAria") : t("waldo.project.detachAria")} className="rounded-md px-2 py-1 text-xs text-muted-foreground hover:bg-interactive-hover" disabled={!daemonReady || busy} onClick={() => void detachContext(selectedAttachment)} type="button">
                {t("waldo.project.detach")}
              </button>
            </>
          ) : (
            <button className="inline-flex items-center gap-1 rounded-full border border-border px-2.5 py-1 text-xs text-muted-foreground hover:bg-interactive-hover disabled:opacity-50" disabled={!daemonReady || !snapshot || busy} onClick={() => void attachSelectedContext()} type="button">
              <Link2 aria-hidden="true" className="size-3" /> {outcomeId ? t("waldo.project.attachOutcome") : t("waldo.project.attach")}
            </button>
          )}
        </div>
        {!daemonReady ? <p className="mt-2 text-xs text-warning-foreground">{t("waldo.project.offline")}</p> : null}
        {onOpenHome ? (
          <button aria-label={t("waldo.project.homeAria")} className="mt-2 text-xs font-medium text-muted-foreground underline-offset-4 hover:text-foreground hover:underline" onClick={onOpenHome} type="button">
            {t("waldo.project.home")}
          </button>
        ) : null}
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4">
        {loading ? <p className="text-sm text-muted-foreground">{t("waldo.project.loading")}</p> : null}
        {!loading && !snapshot ? <p className="text-sm text-muted-foreground">{t("waldo.project.noState")}</p> : null}
        {snapshot ? (
          <div aria-label={t("waldo.project.logAria")} className="space-y-3" role="log">
            {snapshot.turns.length === 0 ? <p className="text-sm leading-6 text-muted-foreground">{t("waldo.project.empty")}</p> : null}
            {snapshot.turns.map((turn) => (
              <article className={cn("max-w-[92%] rounded-2xl px-3.5 py-3 text-sm leading-5", turn.role === "user" ? "ml-auto rounded-br-md bg-foreground text-background" : "mr-auto rounded-bl-md border border-border bg-raised text-foreground")} key={turn.id}>
                <span className="sr-only">{t("waldo.project.turn", { sequence: turn.sequence, role: turn.role })}</span>
                {turn.message}
              </article>
            ))}
          </div>
        ) : null}

        {(snapshot?.continuationReceipts.length ?? 0) > 0 ? (
          <section aria-label={t("waldo.project.receiptsAria")} className="mt-5 space-y-2 border-t border-border pt-4">
            {snapshot?.continuationReceipts.map((receipt) => (
              <article className="rounded-xl border border-border bg-card p-3" key={receipt.id}>
                <div className="flex items-center gap-2 text-xs font-semibold text-foreground">
                  {receipt.action === "automatic" ? <CheckCircle2 aria-hidden="true" className="size-3.5 text-success-foreground" /> : <AlertTriangle aria-hidden="true" className="size-3.5 text-warning-foreground" />}
                  {receipt.action === "automatic" ? t("waldo.project.continuedSafely") : t("waldo.project.needsYou")}
                </div>
                <p className="mt-1 text-xs leading-5 text-muted-foreground">{receipt.action === "automatic" ? receipt.reasonDetail : receipt.needsUserReason || receipt.reasonDetail}</p>
                <p className="mt-1 text-micro text-muted-foreground">{receipt.oldSessionFenced ? t("waldo.project.predecessorFenced") : t("waldo.project.fencingUnconfirmed")} · {receipt.replacementIdentityConfirmed ? t("waldo.project.replacementConfirmed") : t("waldo.project.noReplacement")}</p>
              </article>
            ))}
          </section>
        ) : null}

        {error ? (
          <div className="mt-4 rounded-xl border border-destructive/30 bg-destructive/5 p-3 text-xs text-destructive" role="alert">
            {error}{requestId ? ` · Request ${requestId}` : ""}
          </div>
        ) : null}
        {notice ? <p className="mt-3 text-xs leading-5 text-muted-foreground" role="status">{notice}</p> : null}
      </div>

      <div className="border-t border-border p-3.5">
        <label className="sr-only" htmlFor={`waldo-message-${projectId}`}>{t("waldo.project.composerLabel")}</label>
        <textarea id={`waldo-message-${projectId}`} aria-label={t("waldo.project.composerLabel")} className="min-h-20 w-full resize-none rounded-xl border border-border bg-muted/40 px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70" onChange={(event) => setDraft(event.target.value)} onKeyDown={(event) => {
          if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
            event.preventDefault();
            void sendTurn();
          }
        }} placeholder={daemonReady ? t("waldo.project.placeholder") : t("waldo.project.placeholderOffline")} value={draft} />
        <div className="mt-2 flex items-center justify-between gap-3">
          <p className="text-micro leading-4 text-muted-foreground">{activeEpisode ? t("waldo.project.episode", { ordinal: activeEpisode.ordinal }) : t("waldo.project.episodeOnSend")} · {t("waldo.project.shortcut")}</p>
          <button aria-label={t("waldo.project.sendAria")} className="inline-flex h-8 items-center gap-1.5 rounded-lg bg-foreground px-3 text-xs font-medium text-background disabled:opacity-40" disabled={!daemonReady || !snapshot || !draft.trim() || busy} onClick={() => void sendTurn()} type="button">
            <Send aria-hidden="true" className="size-3.5" /> {t("waldo.project.send")}
          </button>
        </div>
        {!daemonReady && draft ? <p className="mt-2 inline-flex items-center gap-1 text-micro text-muted-foreground"><Link2Off aria-hidden="true" className="size-3" /> {t("waldo.project.draftLocal")}</p> : null}
      </div>
    </div>
  );
}
