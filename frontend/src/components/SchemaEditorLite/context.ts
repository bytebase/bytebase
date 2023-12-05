import { InjectionKey, Ref, inject, provide } from "vue";
import { ComposedProject } from "@/types";
import { EditTarget, ResourceType } from "./types";

export type SchemaEditorContext = {
  resourceType: Ref<ResourceType>;
  readonly: Ref<boolean>;
  project: Ref<ComposedProject>;
  targets: Ref<EditTarget[]>;
};

export const KEY = Symbol(
  "bb.schema-editor"
) as InjectionKey<SchemaEditorContext>;

export const useSchemaEditorContext = () => {
  return inject(KEY)!;
};

export const provideSchemaEditorContext = (
  params: Pick<
    SchemaEditorContext,
    "targets" | "resourceType" | "readonly" | "project"
  >
) => {
  const context: SchemaEditorContext = {
    ...params,
  };

  provide(KEY, context);

  return context;
};
