import Emittery from "emittery";
import type { Ref } from "vue";
import { inject, provide, computed } from "vue";
import { supportSetClassificationFromComment } from "@/components/ColumnDataTable/utils";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { DataClassificationSetting_DataClassificationConfig as DataClassificationConfig } from "@/types/proto-es/v1/setting_service_pb";
import type { RebuildMetadataEditReset } from "../algorithm/rebuild";
import type { EditTarget, RolloutObject } from "../types";
import { useEditCatalogs } from "./config";
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
  classificationConfig: Ref<DataClassificationConfig | undefined>;
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
    ...useEditCatalogs(
      computed(() =>
        params.targets.value.map((target) => ({
          database: target.database.name,
          catalog: target.catalog,
        }))
      )
    ),
    ...useScrollStatus(),
    ...useSelection(params.selectedRolloutObjects, events),
    showClassificationColumn: (
      engine: Engine,
      classificationFromConfig: boolean
    ) => {
      return supportSetClassificationFromComment(
        engine,
        classificationFromConfig
      );
    },
  };

  provide(KEY, context);

  return context;
};

export type SchemaEditorContext = ReturnType<typeof provideSchemaEditorContext>;

export const useSchemaEditorContext = () => {
  return inject(KEY) as SchemaEditorContext;
};
