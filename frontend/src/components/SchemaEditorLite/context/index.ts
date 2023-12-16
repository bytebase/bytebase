import Emittery from "emittery";
import { Ref, inject, provide } from "vue";
import { ComposedProject } from "@/types";
import { EditTarget, ResourceType, RolloutObject } from "../types";
import { useEditConfigs } from "./config";
import { useEditStatus } from "./edit";
import { useTabs } from "./tabs";

export const KEY = Symbol("bb.schema-editor");

export type SchemaEditorEvents = Emittery<{
  ["update:selected-rollout-objects"]: RolloutObject[];
}>;

export const provideSchemaEditorContext = (params: {
  resourceType: Ref<ResourceType>;
  readonly: Ref<boolean>;
  project: Ref<ComposedProject>;
  targets: Ref<EditTarget[]>;
  selectedRolloutObjects: Ref<RolloutObject[] | undefined>;
}) => {
  const context = {
    events: new Emittery() as SchemaEditorEvents,
    ...params,
    ...useTabs(),
    ...useEditStatus(),
    ...useEditConfigs(params.targets),
  };

  provide(KEY, context);

  return context;
};

export type SchemaEditorContext = ReturnType<typeof provideSchemaEditorContext>;

export const useSchemaEditorContext = () => {
  return inject(KEY) as SchemaEditorContext;
};
