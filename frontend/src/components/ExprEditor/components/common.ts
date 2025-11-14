import type { ComputedRef, Ref } from "vue";
import { computed } from "vue";
import type { ConditionExpr } from "@/plugins/cel";
import {
  type Factor,
  getOperatorListByFactor as getRawOperatorListByFactor,
  type Operator,
} from "@/plugins/cel";
import { t, te } from "@/plugins/i18n";
import { type OptionConfig, useExprEditorContext } from "../context";

export const useSelectOptionConfig = (
  expr: Ref<ConditionExpr>
): {
  optionConfig: ComputedRef<OptionConfig>;
  factor: ComputedRef<string>;
} => {
  const context = useExprEditorContext();
  const { optionConfigMap } = context;

  const factor = computed(() => expr.value.args[0]);

  const optionConfig = computed(() => {
    return (
      optionConfigMap.value.get(factor.value) || { remote: false, options: [] }
    );
  });

  return { optionConfig, factor };
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
