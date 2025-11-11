export const SheetViewModeList = ["my", "shared"] as const;
export type SheetViewMode = (typeof SheetViewModeList)[number];
