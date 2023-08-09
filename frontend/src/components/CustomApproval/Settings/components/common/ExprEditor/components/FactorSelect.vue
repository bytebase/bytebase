<template>
  <NSelect
    v-model:value="factor"
    :options="options"
    :consistent-menu-width="false"
    :disabled="!allowAdmin"
    size="small"
    style="width: auto; max-width: 9rem; flex-shrink: 0"
  />
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */
import { SelectOption } from "naive-ui";
import { computed, watch } from "vue";
import {
  type ConditionExpr,
  type Factor,
  isHighLevelFactor,
} from "@/plugins/cel";
import { Risk_Source } from "@/types/proto/v1/risk_service";
import { factorText } from "../../utils";
import { useExprEditorContext } from "../context";
import { FactorList } from "../factor";

const props = defineProps<{
  expr: ConditionExpr;
}>();

const { allowAdmin, allowHighLevelFactors, riskSource } =
  useExprEditorContext();

const factor = computed({
  get() {
    return props.expr.args[0];
  },
  set(factor) {
    props.expr.args[0] = factor;
  },
});

const factorList = computed((): Factor[] => {
  const factorList: Factor[] = [];
  const source = riskSource.value;
  switch (source) {
    case Risk_Source.DDL:
      factorList.push(...FactorList.DDL);
      break;
    case Risk_Source.DML:
      factorList.push(...FactorList.DML);
      break;
    case Risk_Source.CREATE_DATABASE:
      factorList.push(...FactorList.CreateDatabase);
      break;
    case Risk_Source.QUERY:
      factorList.push(...FactorList.RequestQuery);
      break;
    case Risk_Source.EXPORT:
      factorList.push(...FactorList.RequestExport);
      break;
    default:
      // unsupported namespace
      return [];
  }
  if (allowHighLevelFactors.value) return factorList;
  return factorList.filter((factor) => !isHighLevelFactor(factor));
});

const options = computed(() => {
  return factorList.value.map<SelectOption>((v) => ({
    label: factorText(v),
    value: v,
  }));
});

watch(
  [factor, factorList],
  () => {
    if (factorList.value.length === 0) return;
    if (!factorList.value.includes(factor.value)) {
      factor.value = factorList.value[0];
    }
  },
  { immediate: true }
);
</script>
