import {
  createContext,
  createRef,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
  type RefObject,
} from "react";
import { matchesRendererShortcut } from "../../stores/keybindings-store";

export type WaldoPreviewMode = "conversation" | "activity";
export type WaldoPreviewEpisode = "fresh" | "contextual" | "returning";

export type WaldoConversationState = {
  contextDetached: boolean;
  draft: string;
  episode: WaldoPreviewEpisode;
  mode: WaldoPreviewMode;
  selectEpisode: (episode: WaldoPreviewEpisode) => void;
  setContextDetached: (detached: boolean) => void;
  setDraft: (draft: string) => void;
  setMode: (mode: WaldoPreviewMode) => void;
};

type WaldoRailContextValue = {
  isOpen: boolean;
  launcherRef: RefObject<HTMLButtonElement | null>;
  open: (origin?: HTMLElement | null) => void;
  close: () => void;
  toggle: (origin?: HTMLElement | null) => void;
  approvalActive: boolean;
  setApprovalActive: (active: boolean) => void;
  /** Shared relationship state. Presentation components may fall back to local
   * state when rendered outside the shell provider in isolated tests. */
  conversation?: WaldoConversationState;
};

const fallbackLauncherRef = createRef<HTMLButtonElement>();

const fallbackValue: WaldoRailContextValue = {
  isOpen: false,
  launcherRef: fallbackLauncherRef,
  open: () => undefined,
  close: () => undefined,
  toggle: () => undefined,
  approvalActive: false,
  setApprovalActive: () => undefined,
};

const WaldoRailContext = createContext<WaldoRailContextValue>(fallbackValue);

export function WaldoRailProvider({ children }: { children: ReactNode }) {
  const [isOpen, setIsOpen] = useState(false);
  const [approvalActive, setApprovalActive] = useState(false);
  const [mode, setMode] = useState<WaldoPreviewMode>("conversation");
  const [episode, setEpisode] = useState<WaldoPreviewEpisode>("contextual");
  const [contextDetached, setContextDetached] = useState(false);
  const [draft, setDraft] = useState("");
  const invocationOriginRef = useRef<HTMLElement | null>(null);
  const launcherRef = useRef<HTMLButtonElement>(null);

  const open = useCallback((origin?: HTMLElement | null) => {
    invocationOriginRef.current =
      origin ?? (document.activeElement instanceof HTMLElement ? document.activeElement : null);
    setIsOpen(true);
  }, []);

  const close = useCallback(() => {
    const origin = invocationOriginRef.current;
    invocationOriginRef.current = null;
    setApprovalActive(false);
    setIsOpen(false);
    window.requestAnimationFrame(() => {
      const target = origin?.isConnected ? origin : launcherRef.current;
      target?.focus({ preventScroll: true });
    });
  }, []);

  const toggle = useCallback(
    (origin?: HTMLElement | null) => {
      if (isOpen) {
        close();
        return;
      }
      open(origin);
    },
    [close, isOpen, open],
  );

  const selectEpisode = useCallback((nextEpisode: WaldoPreviewEpisode) => {
    setEpisode(nextEpisode);
    setContextDetached(false);
  }, []);

  useEffect(() => {
    if (!isOpen) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape" || event.defaultPrevented || approvalActive) return;
      event.preventDefault();
      close();
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [approvalActive, close, isOpen]);

  const value = useMemo<WaldoRailContextValue>(
    () => ({
      approvalActive,
      close,
      conversation: {
        contextDetached,
        draft,
        episode,
        mode,
        selectEpisode,
        setContextDetached,
        setDraft,
        setMode,
      },
      isOpen,
      launcherRef,
      open,
      setApprovalActive,
      toggle,
    }),
    [approvalActive, close, contextDetached, draft, episode, isOpen, mode, open, selectEpisode, toggle],
  );

  return <WaldoRailContext.Provider value={value}>{children}</WaldoRailContext.Provider>;
}

export function useWaldoRail() {
  return useContext(WaldoRailContext);
}

export function WaldoShortcutRuntime() {
  const waldo = useWaldoRail();

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (!matchesRendererShortcut("toggle-waldo", event)) return;
      event.preventDefault();
      waldo.toggle(
        document.activeElement instanceof HTMLElement ? document.activeElement : null,
      );
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [waldo]);

  return null;
}
