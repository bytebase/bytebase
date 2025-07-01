import Emittery from "emittery";
import type { InjectionKey, Ref } from "vue";
import { inject, provide, ref } from "vue";
import type { Changelist_Change_Source as ChangeSource } from "@/types";
import type { Changelist_Change as Change } from "@/types/proto-es/v1/changelist_service_pb";
import { useChangelistDetailContext } from "../context";
import { emptyRawSQLChange } from "./utils";

// eslint-disable-next-line @typescript-eslint/no-empty-object-type
export type AddChangeEvents = Emittery<{}>;

export type AddChangeContext = {
  changeSource: Ref<ChangeSource>;
  changesFromChangelog: Ref<Change[]>;
  changeFromRawSQL: Ref<Change>;

  events: AddChangeEvents;
};

export const KEY = Symbol(
  "bb.changelist.add-change"
) as InjectionKey<AddChangeContext>;

export const useAddChangeContext = () => {
  return inject(KEY)!;
};

export const provideAddChangeContext = () => {
  const { project } = useChangelistDetailContext();

  const context: AddChangeContext = {
    changeSource: ref("CHANGELOG"),
    changesFromChangelog: ref([]),
    changeFromRawSQL: ref(emptyRawSQLChange(project.value.name)),

    events: new Emittery(),
  };

  provide(KEY, context);

  return context;
};
