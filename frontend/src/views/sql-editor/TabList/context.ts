import Emittery from "emittery";
import type { InjectionKey, Ref } from "vue";
import { inject, provide, ref } from "vue";
import type { SQLEditorTab } from "@/types";

export type CloseTabAction =
  | "CLOSE"
  | "CLOSE_OTHERS"
  | "CLOSE_TO_THE_RIGHT"
  | "CLOSE_SAVED"
  | "CLOSE_ALL";

type TabListEvents = Emittery<{
  "close-tab": { tab: SQLEditorTab; index: number; action: CloseTabAction };
  "rename-tab": { tab: SQLEditorTab; index: number };
}>;

export type TabListContextMenuState = {
  x: number;
  y: number;
  tab: SQLEditorTab;
  index: number;
};

export type TabListContext = {
  contextMenu: Ref<TabListContextMenuState | undefined>;
  events: TabListEvents;
};

export const KEY = Symbol(
  "bb.sql-editor.tab-list"
) as InjectionKey<TabListContext>;

export const useTabListContext = () => {
  return inject(KEY)!;
};

export const provideTabListContext = () => {
  const context: TabListContext = {
    contextMenu: ref(),
    events: new Emittery(),
  };

  provide(KEY, context);

  return context;
};
