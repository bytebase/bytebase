import Emittery from "emittery";
import { InjectionKey, Ref, inject, provide, ref } from "vue";
import { Changelist_Change_Source as ChangeSource } from "@/types";
import { Changelist_Change } from "@/types/proto/v1/changelist_service";

export type AddChangeEvents = Emittery<{
  // not used yet
}>;

export type AddChangeContext = {
  changeSource: Ref<ChangeSource>;
  changesFromChangeHistory: Ref<Changelist_Change[]>;
  changesFromBranch: Ref<Changelist_Change[]>;
  changesFromRawSQL: Ref<Changelist_Change[]>;

  events: AddChangeEvents;
};

export const KEY = Symbol(
  "bb.changelist.add-change"
) as InjectionKey<AddChangeContext>;

export const useAddChangeContext = () => {
  return inject(KEY)!;
};

export const provideAddChangeContext = () => {
  const context: AddChangeContext = {
    changeSource: ref("CHANGE_HISTORY"),
    changesFromChangeHistory: ref([]),
    changesFromBranch: ref([]),
    changesFromRawSQL: ref([]),

    events: new Emittery(),
  };

  provide(KEY, context);

  return context;
};
