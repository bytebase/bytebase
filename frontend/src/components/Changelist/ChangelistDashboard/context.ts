import Emittery from "emittery";
import type { InjectionKey, Ref } from "vue";
import { inject, provide, ref } from "vue";

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

export const provideChangelistDashboardContext = (
  project: Ref<string> = ref("projects/-")
) => {
  const context: ChangelistDashboardContext = {
    filter: ref({
      project,
      keyword: "",
    }),
    showCreatePanel: ref(false),
    events: new Emittery(),
  };

  provide(KEY, context);

  return context;
};
