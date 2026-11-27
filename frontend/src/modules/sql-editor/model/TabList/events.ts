import Emittery from "emittery";
import type { SQLEditorTab } from "@/types";

export type CloseTabAction =
  | "CLOSE"
  | "CLOSE_OTHERS"
  | "CLOSE_TO_THE_RIGHT"
  | "CLOSE_SAVED"
  | "CLOSE_ALL";

export type TabListEventMap = {
  "close-tab": { tab: SQLEditorTab; index: number; action: CloseTabAction };
  "rename-tab": { tab: SQLEditorTab; index: number };
};

export const tabListEvents: Emittery<TabListEventMap> = new Emittery();
