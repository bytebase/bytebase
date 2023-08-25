import { type Ref, computed } from "vue";
import { ConditionExpr, type Factor } from "@/plugins/cel";
import { t, te } from "@/plugins/i18n";
import { useExprEditorContext } from "../context";

export const useSelectOptions = (expr: Ref<ConditionExpr>) => {
  const context = useExprEditorContext();
  const { factorOptionsMap } = context;

  const options = computed(() => {
    const factor = expr.value.args[0];
    return factorOptionsMap.value.get(factor) || [];
  });

  return options;
};

export const factorText = (factor: Factor) => {
  const keypath = `cel.factor.${factor}`;
  if (te(keypath)) {
    return t(keypath);
  }
  return factor;
};
