<!-- eslint-disable vue/no-mutating-props -->
<template>
  <NSelect
    v-model:value="operator"
    :options="options"
    :consistent-menu-width="false"
    :disabled="!allowAdmin"
    size="small"
    style="width: auto; max-width: 7rem; min-width: 2.5rem; flex-shrink: 0"
  />
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */
import { SelectOption } from "naive-ui";
import { computed, watch } from "vue";
import {
  type Operator,
  type ConditionOperator,
  type ConditionExpr,
  getOperatorListByFactor,
} from "@/plugins/cel";
import { useExprEditorContext } from "../context";

const props = defineProps<{
  expr: ConditionExpr;
}>();

const context = useExprEditorContext();
const { allowAdmin } = context;

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

const OPERATOR_DICT = new Map([
  ["_==_", "=="],
  ["_!=_", "!="],
  ["_<_", "<"],
  ["_<=_", "≤"],
  ["_>=_", "≥"],
  ["_>_", ">"],
]);

const options = computed(() => {
  const operators = getOperatorListByFactor(factor.value);

  const mapOption = (op: Operator): SelectOption => {
    const label = OPERATOR_DICT.get(op) ?? op.replace(/^@/g, "");
    return {
      label,
      value: op,
    };
  };
  return operators.map(mapOption);
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
