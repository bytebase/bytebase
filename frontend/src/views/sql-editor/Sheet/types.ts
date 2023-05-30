export const SheetViewModeList = ["my", "shared", "starred"] as const;
export type SheetViewMode = typeof SheetViewModeList[number];
