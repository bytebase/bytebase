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
 * Module-level tab list event bus. Imported by both the Vue
 * `provideTabListContext` (via `./context`) and React consumers that can't
 * participate in Vue's provide/inject. Single shared instance so emit/on
 * from either side reaches the other — same pattern used for the AI events
 * singleton (see `src/plugins/ai/logic/events.ts`).
 */
export const tabListEvents: Emittery<TabListEventMap> = new Emittery();
