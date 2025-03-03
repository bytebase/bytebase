import type { SelectOption } from "naive-ui";
import { inject, provide, type InjectionKey, type Ref } from "vue";
import type { Factor, Operator } from "@/plugins/cel";

export type OptionConfig = {
  remote: boolean;
  search?: (keyword: string) => Promise<SelectOption[]>;
  options: SelectOption[];
};

export type ExprEditorContext = {
  allowAdmin: Ref<boolean>;
  enableRawExpression: Ref<boolean>;
  factorList: Ref<Factor[]>;
  factorSupportDropdown: Ref<Factor[]>;
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
