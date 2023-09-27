import Emittery from "emittery";
import { InjectionKey, Ref, inject, provide, ref } from "vue";

export type ChangelistsFilter = {
  project: string;
  keyword: string;
};

export type ChangelistsEvents = Emittery<{
  refresh: undefined;
}>;

export type ChangelistsContext = {
  filter: Ref<ChangelistsFilter>;
  showCreatePanel: Ref<boolean>;
  events: ChangelistsEvents;
};

export const KEY = Symbol(
  "bb.changelists.context"
) as InjectionKey<ChangelistsContext>;

export const useChangelistsContext = () => {
  return inject(KEY)!;
};

export const provideChangelistsContext = () => {
  const context: ChangelistsContext = {
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
