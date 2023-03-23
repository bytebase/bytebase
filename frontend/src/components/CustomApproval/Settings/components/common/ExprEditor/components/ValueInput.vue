<template>
  <template v-if="inputType === 'INPUT'">
    <NumberInput
      v-if="isNumberValue"
      :value="getNumberValue()"
      @update:value="setNumericValue($event)"
    />
    <StringInput
      v-if="isStringValue"
      :value="getStringValue()"
      @update:value="setStringValue($event)"
    />
  </template>
  <template v-if="inputType === 'SINGLE-SELECT'">
    <SingleSelect
      v-if="isNumberValue"
      :value="getNumberValue()"
      :expr="expr"
      @update:value="setNumericValue($event as number)"
    />
    <SingleSelect
      v-if="isStringValue"
      :value="getStringValue()"
      :expr="expr"
      @update:value="setStringValue($event as string)"
    />
  </template>
  <MultiSelect
    v-if="isArrayValue"
    :value="getArrayValue()"
    :expr="expr"
    @update:value="setArrayValue($event)"
  />
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */

import { computed, watch } from "vue";
import { isNumber } from "lodash-es";

import {
  type ConditionExpr,
  isEqualityOperator,
  isCollectionOperator,
  isStringOperator,
  isCompareOperator,
  isNumberFactor,
  isStringFactor,
} from "@/plugins/cel";
import NumberInput from "./NumberInput.vue";
import MultiSelect from "./MultiSelect.vue";
import StringInput from "./StringInput.vue";
import SingleSelect from "./SingleSelect.vue";

type InputType = "INPUT" | "SINGLE-SELECT" | "MULTI-SELECT";

const props = defineProps<{
  expr: ConditionExpr;
}>();

const operator = computed(() => {
  return props.expr.operator;
});

const factor = computed(() => {
  return props.expr.args[0];
});

const isNumberValue = computed(() => {
  if (isCompareOperator(operator.value)) return true;
  if (isEqualityOperator(operator.value) && isNumberFactor(factor.value)) {
    return true;
  }
  return false;
});
const isStringValue = computed(() => {
  if (isStringOperator(operator.value)) return true;
  if (isEqualityOperator(operator.value) && isStringFactor(factor.value)) {
    return true;
  }
  return false;
});
const isArrayValue = computed(() => {
  return isCollectionOperator(operator.value);
});
const inputType = computed((): InputType => {
  if (isArrayValue.value) return "SINGLE-SELECT";
  if (isEqualityOperator(operator.value)) {
    return "SINGLE-SELECT";
  }
  return "INPUT";
});

const getNumberValue = () => {
  const value = props.expr.args[1];
  if (!isNumber(value)) return 0;
  return value;
};
const setNumericValue = (value: number) => {
  props.expr.args[1] = value;
};

const getStringValue = () => {
  const value = props.expr.args[1];
  if (typeof value !== "string") return "";
  return value;
};
const setStringValue = (value: string) => {
  props.expr.args[1] = value;
};

const getArrayValue = () => {
  const values = props.expr.args[1];
  if (!Array.isArray(values)) return [];
  return values;
};
const setArrayValue = (values: string[] | number[]) => {
  props.expr.args[1] = values;
};

watch(
  [() => props.expr.args[1], () => props.expr.operator],
  ([value, operator]) => {
    if (isCompareOperator(operator) && !isNumber(value)) {
      props.expr.args[1] = 0;
    }
    if (isCollectionOperator(operator) && !Array.isArray(value)) {
      props.expr.args[1] = [];
    }
    if (isStringOperator(operator) && typeof value !== "string") {
      props.expr.args[1] = "";
    }
  },
  { immediate: true }
);
</script>
