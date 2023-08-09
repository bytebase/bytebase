import { inject, provide, type InjectionKey, type Ref } from "vue";
import { Risk_Source } from "@/types/proto/v1/risk_service";

export type ExprEditorContext = {
  allowAdmin: Ref<boolean>;
  allowHighLevelFactors: Ref<boolean>;
  riskSource: Ref<Risk_Source | undefined>;
};

export const KEY = Symbol(
  "bb.custom-approval.expr-editor"
) as InjectionKey<ExprEditorContext>;

export const useExprEditorContext = () => {
  return inject(KEY)!;
};

export const provideExprEditorContext = (context: ExprEditorContext) => {
  provide(KEY, context);
};
