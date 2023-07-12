import { inject, InjectionKey, provide, Ref } from "vue";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";

interface SchemaDesignerContext {
  metadata: Ref<DatabaseMetadata>;
  baselineMetadata: Ref<DatabaseMetadata>;
}

export const KEY = Symbol(
  "bb.schema-designer"
) as InjectionKey<SchemaDesignerContext>;

export const useSchemaDesignerContext = () => {
  return inject(KEY)!;
};

export const provideSchemaDesignerContext = (
  context: SchemaDesignerContext
) => {
  provide(KEY, context);
};
