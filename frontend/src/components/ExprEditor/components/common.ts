import type { Ref, ComputedRef } from "vue";
import { computed } from "vue";
import type { ConditionExpr } from "@/plugins/cel";
import {
  getOperatorListByFactor as getRawOperatorListByFactor,
  type Factor,
  type Operator,
} from "@/plugins/cel";
import { t, te } from "@/plugins/i18n";
import { useExprEditorContext, type OptionConfig } from "../context";

export const useSelectOptions = (
  expr: Ref<ConditionExpr>
): ComputedRef<OptionConfig> => {
  const context = useExprEditorContext();
  const { optionConfigMap } = context;

  const options = computed(() => {
    const factor = expr.value.args[0];
    return optionConfigMap.value.get(factor) || { remote: false, options: [] };
  });

  return options;
};

const stringifyFactor = (factor: Factor) => {
  return factor.replace(/\./g, "_");
};

export const factorText = (factor: Factor) => {
  const keypath = `cel.factor.${stringifyFactor(factor)}`;
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
