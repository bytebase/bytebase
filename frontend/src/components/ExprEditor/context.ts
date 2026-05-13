import { type InjectionKey, inject, provide, type Ref } from "vue";
import type { Factor, Operator } from "@/plugins/cel";
import type { ResourceSelectOption } from "@/types/v2-shared";

export type OptionConfig = {
  search?: (params: {
    search: string;
    pageToken: string;
    pageSize: number;
  }) => Promise<{
    nextPageToken: string;
    options: ResourceSelectOption<unknown>[];
  }>;
  fetch?: (names: string[]) => Promise<ResourceSelectOption<unknown>[]>;
  fallback?: (value: string) => ResourceSelectOption<unknown>;
  options: ResourceSelectOption<unknown>[];
};

export type ExprEditorContext = {
  readonly: Ref<boolean>;
  enableRawExpression: Ref<boolean>;
  factorList: Ref<Factor[]>;
  optionConfigMap: Ref<Map<Factor, OptionConfig>>;
  factorOperatorOverrideMap: Ref<Map<Factor, Operator[]> | undefined>;
};

export const KEY = Symbol("bb.expr-editor") as InjectionKey<ExprEditorContext>;

export const useExprEditorContext = () => {
  return inject(KEY)!;
};

export const provideExprEditorContext = (context: ExprEditorContext) => {
  provide(KEY, context);
};
