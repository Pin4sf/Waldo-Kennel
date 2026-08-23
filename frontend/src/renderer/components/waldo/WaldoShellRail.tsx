import { WaldoRail } from "./WaldoRail";
import { useWaldoRail } from "./WaldoRailContext";

export function WaldoShellRail({
  contextLabel,
  onReturnToInspector,
  previewEnabled,
}: {
  contextLabel: string;
  onReturnToInspector?: () => void;
  previewEnabled: boolean;
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
        onClose={waldo.close}
        onReturnToInspector={returnToInspector}
        previewEnabled={previewEnabled}
      />
    </div>
  );
}
