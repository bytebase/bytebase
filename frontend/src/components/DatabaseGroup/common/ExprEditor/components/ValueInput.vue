<template>
  <template v-if="inputType === 'INPUT'">
    <StringInput
      v-if="isStringValue"
      :value="getStringValue()"
      @update:value="setStringValue($event)"
    />
  </template>
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */
import { computed, watch } from "vue";
import {
  type ConditionExpr,
  isEqualityOperator,
  isStringOperator,
  isStringFactor,
} from "@/plugins/cel";
import StringInput from "./StringInput.vue";

type InputType = "INPUT";

const props = defineProps<{
  expr: ConditionExpr;
}>();

const operator = computed(() => {
  return props.expr.operator;
});

const factor = computed(() => {
  return props.expr.args[0];
});

const isStringValue = computed(() => {
  if (isStringOperator(operator.value)) return true;
  if (isEqualityOperator(operator.value) && isStringFactor(factor.value)) {
    return true;
  }
  return false;
});
const inputType = computed((): InputType => {
  return "INPUT";
});

const getStringValue = () => {
  const value = props.expr.args[1];
  if (typeof value !== "string") return "";
  return value;
};
const setStringValue = (value: string) => {
  props.expr.args[1] = value;
};

// clean up value type when factor and operator changed
watch(
  [factor, operator, () => props.expr.args[1]],
  ([factor, operator, value]) => {
    if (isEqualityOperator(operator)) {
      if (isStringFactor(factor) && typeof value !== "string") {
        setStringValue("");
      }
    }
    if (isStringOperator(operator) && typeof value !== "string") {
      setStringValue("");
    }
  },
  { immediate: true }
);
</script>
