import { type Ref, computed } from "vue";
import {
  ConditionExpr,
  getOperatorListByFactor as getRawOperatorListByFactor,
  type Factor,
  type Operator,
} from "@/plugins/cel";
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

export const getOperatorListByFactor = (
  factor: Factor,
  factorOperatorOverrideMap: Map<Factor, Operator[]> | undefined
): Operator[] => {
  return (
    factorOperatorOverrideMap?.get(factor) ?? getRawOperatorListByFactor(factor)
  );
};
