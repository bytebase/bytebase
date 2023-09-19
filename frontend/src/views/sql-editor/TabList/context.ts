import Emittery from "emittery";
import { InjectionKey, Ref, inject, provide, ref } from "vue";
import { TabInfo } from "@/types";

export type CloseTabAction =
  | "CLOSE"
  | "CLOSE_OTHERS"
  | "CLOSE_TO_THE_RIGHT"
  | "CLOSE_SAVED"
  | "CLOSE_ALL";

type TabListEvents = Emittery<{
  "close-tab": { tab: TabInfo; index: number; action: CloseTabAction };
  "rename-tab": { tab: TabInfo; index: number };
}>;

export type TabListContextMenuState = {
  x: number;
  y: number;
  tab: TabInfo;
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
