import type { ComputedRef, Ref } from "vue";
import { computed } from "vue";
import type { ResourceSelectOption } from "@/components/v2/Select/RemoteResourceSelector/types";
import type { ConditionExpr } from "@/plugins/cel";
import {
  type Factor,
  getOperatorListByFactor as getRawOperatorListByFactor,
  type Operator,
} from "@/plugins/cel";
import { t, te } from "@/plugins/i18n";
import { getDefaultPagination } from "@/utils";
import { type OptionConfig, useExprEditorContext } from "../context";

export const initOptions = async (
  values: string[],
  optionConfig: OptionConfig
): Promise<ResourceSelectOption<unknown>[]> => {
  if (optionConfig.fetch) {
    return await optionConfig.fetch(values);
  }

  if (optionConfig.search && !optionConfig.fallback) {
    // only use the batch search when not provide the fallback
    const search = optionConfig.search;
    const response = await Promise.all(
      values.map((value) =>
        search({
          search: value,
          pageToken: "",
          pageSize: getDefaultPagination(),
        })
      )
    );
    return response.reduce<ResourceSelectOption<unknown>[]>((list, resp) => {
      list.push(...resp.options);
      return list;
    }, []);
  }

  return [];
};

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
    return optionConfigMap.value.get(factor.value) || { options: [] };
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
