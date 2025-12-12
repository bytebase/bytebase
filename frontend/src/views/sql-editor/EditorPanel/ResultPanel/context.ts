import Emittery from "emittery";
import type { InjectionKey, Ref } from "vue";
import { inject, provide, ref } from "vue";

export type CloseTabAction =
  | "CLOSE"
  | "CLOSE_OTHERS"
  | "CLOSE_TO_THE_RIGHT"
  | "CLOSE_ALL";

type TabListEvents = Emittery<{
  "close-tab": { index: number; action: CloseTabAction };
}>;

export type TabListContextMenuState = {
  x: number;
  y: number;
  index: number;
};

export type TabListContext = {
  contextMenu: Ref<TabListContextMenuState | undefined>;
  events: TabListEvents;
};

export const KEY = Symbol(
  "bb.sql-editor.result-tab-list"
) as InjectionKey<TabListContext>;

export const useResultTabListContext = () => {
  return inject(KEY)!;
};

export const provideResultTabListContext = () => {
  const context: TabListContext = {
    contextMenu: ref(),
    events: new Emittery(),
  };

  provide(KEY, context);

  return context;
};
