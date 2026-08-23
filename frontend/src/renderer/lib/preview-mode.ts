export const usesPreviewWorkspaceData = import.meta.env.VITE_NO_ELECTRON === "1";

export const usesWaldoUiPreview =
  import.meta.env.VITE_NO_ELECTRON === "1" ||
  import.meta.env.VITE_WALDO_UI_PREVIEW === "1";
