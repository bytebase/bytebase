import Emittery from "emittery";
import { InjectionKey, Ref, inject, provide, ref } from "vue";

export type ChangelistDashboardFilter = {
  project: string;
  keyword: string;
};

export type ChangelistDashboardEvents = Emittery<{
  refresh: undefined;
}>;

export type ChangelistDashboardContext = {
  filter: Ref<ChangelistDashboardFilter>;
  showCreatePanel: Ref<boolean>;
  events: ChangelistDashboardEvents;
};

export const KEY = Symbol(
  "bb.changelist.dashboard"
) as InjectionKey<ChangelistDashboardContext>;

export const useChangelistDashboardContext = () => {
  return inject(KEY)!;
};

export const provideChangelistDashboardContext = () => {
  const context: ChangelistDashboardContext = {
    filter: ref({
      project: "projects/-",
      keyword: "",
    }),
    showCreatePanel: ref(false),
    events: new Emittery(),
  };

  provide(KEY, context);

  return context;
};
