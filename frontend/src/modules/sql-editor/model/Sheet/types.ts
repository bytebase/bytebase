export const SheetViewModeList = ["my", "shared", "draft"] as const;
export type SheetViewMode = (typeof SheetViewModeList)[number];
