import Emittery from "emittery";
import type { Ref } from "vue";
import { inject, provide } from "vue";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { RebuildMetadataEditReset } from "../algorithm/rebuild";
import type { EditTarget, RolloutObject } from "../types";
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
  ["refresh-preview"]: undefined;
  ["merge-metadata"]: DatabaseMetadata[];
}>;

export type SchemaEditorOptions = {
  forceShowIndexes: boolean;
  forceShowPartitions: boolean;
};

export const provideSchemaEditorContext = (params: {
  readonly: Ref<boolean>;
  project: Ref<Project>;
  targets: Ref<EditTarget[]>;
  selectedRolloutObjects: Ref<RolloutObject[] | undefined>;
  hidePreview: Ref<boolean>;
  options?: Ref<SchemaEditorOptions>;
}) => {
  const events = new Emittery() as SchemaEditorEvents;

  const context = {
    events,
    ...params,
    ...useTabs(events),
    ...useEditStatus(),
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
