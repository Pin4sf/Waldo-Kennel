export type IslandVisibilityState = {
	supported: boolean;
	enabled: boolean;
	visible: boolean;
	shortcut: string;
};

export const ISLAND_GET_STATE_CHANNEL = "island:getState";
export const ISLAND_SET_VISIBLE_CHANNEL = "island:setVisible";
export const ISLAND_OPEN_SETTINGS_CHANNEL = "island:openSettings";
export const ISLAND_STATE_CHANNEL = "island:state";
