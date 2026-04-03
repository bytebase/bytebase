<!-- eslint-disable vue/no-mutating-props -->
<template>
  <NSelect
    v-model:value="operator"
    :options="options"
    :consistent-menu-width="false"
    :disabled="readonly"
    size="small"
    style="width: auto; max-width: 7rem; min-width: 2.5rem; shrink: 0"
  />
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */
import { NSelect, type SelectOption } from "naive-ui";
import { computed, watch } from "vue";
import {
  type ConditionExpr,
  type ConditionOperator,
  type Operator,
  operatorDisplayLabel,
} from "@/plugins/cel";
import { useExprEditorContext } from "../context";
import { getOperatorListByFactor } from "./common";

const props = defineProps<{
  expr: ConditionExpr;
}>();

const context = useExprEditorContext();
const { readonly, factorOperatorOverrideMap } = context;

const operator = computed({
  get() {
    return props.expr.operator;
  },
  set(op) {
    props.expr.operator = op;
  },
});

const factor = computed(() => {
  return props.expr.args[0];
});

const options = computed(() => {
  const operators = getOperatorListByFactor(
    factor.value,
    factorOperatorOverrideMap.value
  );
  return operators.map(
    (op: Operator): SelectOption => ({
      label: operatorDisplayLabel(op),
      value: op,
    })
  );
});

// normalize operator when factor changed
watch(
  [options, () => props.expr.operator],
  () => {
    if (options.value.length === 0) return;
    if (!options.value.find((opt) => opt.value === props.expr.operator)) {
      props.expr.operator = options.value[0].value as ConditionOperator;
    }
  },
  { immediate: true }
);
</script>
