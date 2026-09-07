import type { CSSProperties } from "react";
import { useTranslation } from "react-i18next";
import { SquareTerminal } from "lucide-react";
import { useUiStore } from "../stores/ui-store";
import { TopbarButton } from "./TopbarButton";
import { Tooltip, TooltipContent, TooltipTrigger } from "./ui/tooltip";

export function NewShellTerminalButton({ style }: { style?: CSSProperties }) {
  const { t } = useTranslation();
  const requestNewShellTerminal = useUiStore(
    (state) => state.requestNewShellTerminal,
  );
  const label = t("shortcut.new-shell-terminal");

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <TopbarButton
          aria-label={label}
          data-priority="secondary"
          onClick={requestNewShellTerminal}
          style={style}
        >
          <SquareTerminal aria-hidden="true" className="size-icon-md" />
        </TopbarButton>
      </TooltipTrigger>
      <TooltipContent side="bottom">{label}</TooltipContent>
    </Tooltip>
  );
}
