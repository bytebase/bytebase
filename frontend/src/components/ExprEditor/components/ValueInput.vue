<template>
  <template v-if="inputType === 'INPUT'">
    <NumberInput
      v-if="isNumberValue"
      :value="getNumberValue()"
      @update:value="setNumberValue($event)"
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
      @update:value="setNumberValue($event as number)"
    />
    <SingleSelect
      v-if="isStringValue"
      :value="getStringValue()"
      :expr="expr"
      @update:value="setStringValue($event as string)"
    />
  </template>
  <template v-if="inputType === 'MULTI-SELECT'">
    <MultiSelect
      :value="getArrayValue()"
      :expr="expr"
      @update:value="setArrayValue($event)"
    />
  </template>
  <template v-if="inputType === 'MULTI-INPUT'">
    <MultiStringInput
      :value="getStringArrayValue()"
      :expr="expr"
      @update:value="setArrayValue($event)"
    />
  </template>
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */
import { isNumber } from "lodash-es";
import { computed, watch } from "vue";
import {
  type ConditionExpr,
  isEqualityOperator,
  isCollectionOperator,
  isStringOperator,
  isCompareOperator,
  isNumberFactor,
  isStringFactor,
} from "@/plugins/cel";
import { useExprEditorContext } from "../context";
import MultiSelect from "./MultiSelect.vue";
import MultiStringInput from "./MultiStringInput.vue";
import NumberInput from "./NumberInput.vue";
import SingleSelect from "./SingleSelect.vue";
import StringInput from "./StringInput.vue";

type InputType = "INPUT" | "SINGLE-SELECT" | "MULTI-SELECT" | "MULTI-INPUT";

const props = defineProps<{
  expr: ConditionExpr;
}>();

const operator = computed(() => {
  return props.expr.operator;
});

const factor = computed(() => {
  return props.expr.args[0];
});

const { factorSupportDropdown } = useExprEditorContext();

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
  if (isArrayValue.value) {
    return factorSupportDropdown.value.includes(factor.value)
      ? "MULTI-SELECT"
      : "MULTI-INPUT";
  }
  if (isEqualityOperator(operator.value)) {
    if (factorSupportDropdown.value.includes(factor.value)) {
      return "SINGLE-SELECT";
    }
  }
  return "INPUT";
});

const getNumberValue = () => {
  const value = props.expr.args[1];
  if (!isNumber(value)) return 0;
  return value;
};
const setNumberValue = (value: number) => {
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
const getStringArrayValue = () => getArrayValue() as string[];
const setArrayValue = (values: string[] | number[]) => {
  props.expr.args[1] = values;
};

// clean up value type when factor and operator changed
watch(
  [factor, operator, () => props.expr.args[1]],
  ([factor, operator, value]) => {
    if (isEqualityOperator(operator)) {
      if (isNumberFactor(factor) && !isNumber(value)) {
        setNumberValue(0);
      }
      if (isStringFactor(factor) && typeof value !== "string") {
        setStringValue("");
      }
    }
    if (isCompareOperator(operator) && !isNumber(value)) {
      setNumberValue(0);
    }
    if (isCollectionOperator(operator) && !Array.isArray(value)) {
      setArrayValue([]);
    }
    if (isStringOperator(operator) && typeof value !== "string") {
      setStringValue("");
    }
  },
  { immediate: true }
);
</script>
