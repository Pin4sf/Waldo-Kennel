import type { TFunction } from "i18next";

export type ReasoningStatusLike = {
  provider: string;
  configured: boolean;
  ready: boolean;
  verified: boolean;
  errorCode?: string;
};

/**
 * Render only stable, allowlisted readiness states. The daemon's free-form
 * detail is deliberately not treated as renderer-safe content; the stable
 * code is enough to tell the owner what to do next.
 */
export function reasoningStatusMessage(
  status: ReasoningStatusLike | undefined,
  draftProvider: string,
  t: TFunction,
): string {
  if (!status || !status.provider) return t("settings.reasoning.missing");
  if (status.provider !== draftProvider)
    return t("settings.reasoning.draftChanged");

  switch (status.errorCode) {
    case "MISSING_CREDENTIAL":
      return t("settings.reasoning.missing");
    case "CREDENTIAL_REJECTED":
    case "AUTH_REQUIRED":
      return t("settings.reasoning.credentialRejected");
    case "PROVIDER_NOT_READY":
    case "REASONING_NOT_READY":
      if (status.provider === "codex")
        return t("onboarding.agent.reasoningNotReady");
      return t("settings.reasoning.missing");
    case "REASONING_RATE_LIMITED":
    case "REASONING_TIMED_OUT":
    case "REASONING_CANCELLED":
    case "REASONING_DECLINED":
    case "REASONING_INCOMPLETE":
    case "REASONING_INVALID_OUTPUT":
    case "REASONING_UNAVAILABLE":
      return t("settings.reasoning.unavailable");
    default:
      break;
  }

  if (!status.configured || !status.ready)
    return t("settings.reasoning.missing");
  return status.verified
    ? t("settings.reasoning.verified")
    : t("settings.reasoning.unverified");
}

/** Whether the saved native Codex provider is unavailable right now. */
export function nativeReasoningUnavailable(
  status: ReasoningStatusLike | undefined,
): boolean {
  return (
    status?.provider === "codex" &&
    !status.ready &&
    (status.errorCode === "PROVIDER_NOT_READY" ||
      status.errorCode === "REASONING_NOT_READY")
  );
}

/** Map durable intake failure codes to owner-safe, localized guidance. */
export function intakeFailureMessage(
  failureCode: string | undefined,
  t: TFunction,
): string | undefined {
  switch (failureCode) {
    case "INTAKE_ANALYSIS_FAILED":
    case "INTAKE_ANALYSIS_INTERRUPTED":
    case "INTAKE_ANALYSIS_INVALID":
      return t("outcome.intake.reasoningFailedBody");
    default:
      return undefined;
  }
}
