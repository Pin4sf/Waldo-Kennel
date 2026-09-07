import { useCallback, useSyncExternalStore } from "react";
import type { IslandAction, IslandModel, KennelIslandAdapter } from "./types";

export function useKennelIsland(adapter: KennelIslandAdapter) {
  const model = useSyncExternalStore(
    adapter.subscribe,
    adapter.getSnapshot,
    adapter.getSnapshot,
  );
  const dispatch = useCallback(
    (action: IslandAction) => adapter.dispatch(action),
    [adapter],
  );

  return { model, dispatch };
}

export interface MutableIslandAdapter extends KennelIslandAdapter {
  replaceSnapshot: (model: IslandModel) => void;
}

export function createMemoryIslandAdapter(
  initialModel: IslandModel,
  reduce: (model: IslandModel, action: IslandAction) => IslandModel,
): MutableIslandAdapter {
  let model = initialModel;
  const listeners = new Set<() => void>();

  const publish = (nextModel: IslandModel) => {
    if (Object.is(nextModel, model)) return;
    model = nextModel;
    listeners.forEach((listener) => listener());
  };

  return {
    getSnapshot: () => model,
    subscribe: (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    dispatch: (action) => publish(reduce(model, action)),
    replaceSnapshot: publish,
  };
}
