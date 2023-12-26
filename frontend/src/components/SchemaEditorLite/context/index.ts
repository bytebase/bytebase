import Emittery from "emittery";
import { Ref, inject, provide } from "vue";
import { ComposedProject } from "@/types";
import { RebuildMetadataEditReset } from "../algorithm/rebuild";
import { EditTarget, ResourceType, RolloutObject } from "../types";
import { useEditConfigs } from "./config";
import { useEditStatus } from "./edit";
import { useScrollStatus } from "./scroll";
import { useSelection } from "./selection";
import { useTabs } from "./tabs";

export const KEY = Symbol("bb.schema-editor");

export type SchemaEditorEvents = Emittery<{
  ["update:selected-rollout-objects"]: RolloutObject[];
  ["rebuild-tree"]: {
    openFirstChild: boolean;
  };
  ["rebuild-edit-status"]: { resets: RebuildMetadataEditReset[] };
  ["clear-tabs"]: undefined;
}>;

export const provideSchemaEditorContext = (params: {
  resourceType: Ref<ResourceType>;
  readonly: Ref<boolean>;
  project: Ref<ComposedProject>;
  targets: Ref<EditTarget[]>;
  selectedRolloutObjects: Ref<RolloutObject[] | undefined>;
}) => {
  const events = new Emittery() as SchemaEditorEvents;
  const context = {
    events,
    ...params,
    ...useTabs(events),
    ...useEditStatus(),
    ...useEditConfigs(params.targets),
    ...useScrollStatus(),
    ...useSelection(params.selectedRolloutObjects, events),
  };

  provide(KEY, context);

  return context;
};

export type SchemaEditorContext = ReturnType<typeof provideSchemaEditorContext>;

export const useSchemaEditorContext = () => {
  return inject(KEY) as SchemaEditorContext;
};
