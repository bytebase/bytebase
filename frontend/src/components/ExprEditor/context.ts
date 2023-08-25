import type { SelectOption } from "naive-ui";
import { inject, provide, type InjectionKey, type Ref } from "vue";
import { type Factor } from "@/plugins/cel";

export type ExprEditorContext = {
  allowAdmin: Ref<boolean>;
  factorList: Ref<Factor[]>;
  factorSupportDropdown: Ref<Factor[]>;
  factorOptionsMap: Ref<Map<Factor, SelectOption[]>>;
};

export const KEY = Symbol("bb.expr-editor") as InjectionKey<ExprEditorContext>;

export const useExprEditorContext = () => {
  return inject(KEY)!;
};

export const provideExprEditorContext = (context: ExprEditorContext) => {
  provide(KEY, context);
};
