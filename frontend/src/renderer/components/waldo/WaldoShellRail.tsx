import { WaldoRail } from "./WaldoRail";
import { useWaldoRail } from "./WaldoRailContext";

export function WaldoShellRail({
  contextLabel,
  daemonReady,
  onOpenHome,
  onReturnToInspector,
  outcomeId,
  outcomeTitle,
  previewEnabled,
  projectId,
  projectName,
}: {
  contextLabel: string;
  daemonReady?: boolean;
  onOpenHome?: () => void;
  onReturnToInspector?: () => void;
  outcomeId?: string;
  outcomeTitle?: string;
  previewEnabled: boolean;
  projectId?: string;
  projectName?: string;
}) {
  const waldo = useWaldoRail();
  if (!waldo.isOpen) return null;

  const returnToInspector = onReturnToInspector
    ? () => {
        onReturnToInspector();
        waldo.close();
      }
    : undefined;

  return (
    <div className="waldo-native-interactive waldo-shell-rail" data-testid="waldo-shell-rail">
      <WaldoRail
        contextLabel={contextLabel}
        daemonReady={daemonReady}
        onClose={waldo.close}
        onOpenHome={onOpenHome}
        onReturnToInspector={returnToInspector}
        outcomeId={outcomeId}
        outcomeTitle={outcomeTitle}
        previewEnabled={previewEnabled}
        projectId={projectId}
        projectName={projectName}
      />
    </div>
  );
}
