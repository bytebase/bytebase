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

/**
 * Module-level tab list event bus. Single shared instance so emit/on from
 * any component reaches the others — same pattern used for the AI events
 * singleton (see `src/modules/ai/logic/events.ts`).
 */
export const tabListEvents: Emittery<TabListEventMap> = new Emittery();
