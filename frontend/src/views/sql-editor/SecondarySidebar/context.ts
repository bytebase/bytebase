import Emittery from "emittery";
import { InjectionKey, inject, provide, Ref, ref } from "vue";

type SecondarySidebarEvents = Emittery<{
  // nothing by now
}>;

export type TabView = "INFO" | "SHEET" | "HISTORY";

export type SecondarySidebarContext = {
  show: Ref<boolean>; // Whether to show the panel
  tab: Ref<TabView>;

  events: SecondarySidebarEvents;
};

export const KEY = Symbol(
  "bb.sql-editor.secondary-sidebar"
) as InjectionKey<SecondarySidebarContext>;

export const useSecondarySidebarContext = () => {
  return inject(KEY)!;
};

export const provideSecondarySidebarContext = () => {
  const context: SecondarySidebarContext = {
    show: ref(true),
    tab: ref("INFO"),
    events: new Emittery(),
  };

  provide(KEY, context);

  return context;
};
