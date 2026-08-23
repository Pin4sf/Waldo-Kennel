import { WaldoRail } from "../waldo/WaldoRail";

export function HomeChat({
  contextLabel,
  previewEnabled,
}: {
  contextLabel: string;
  previewEnabled: boolean;
}) {
  return (
    <WaldoRail
      contextLabel={contextLabel}
      presentation="destination"
      previewEnabled={previewEnabled}
    />
  );
}
