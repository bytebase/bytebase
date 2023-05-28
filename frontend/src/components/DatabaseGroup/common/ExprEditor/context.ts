import { inject, provide, type InjectionKey, type Ref } from "vue";

export type ResourceType = "DATABASE_GROUP" | "SCHEMA_GROUP";

export type ExprEditorContext = {
  allowAdmin: Ref<boolean>;
  resourceType: Ref<ResourceType>;
};

export const KEY = Symbol(
  "bb.database-group.expr-editor"
) as InjectionKey<ExprEditorContext>;

export const useExprEditorContext = () => {
  return inject(KEY)!;
};

export const provideExprEditorContext = (context: ExprEditorContext) => {
  provide(KEY, context);
};
