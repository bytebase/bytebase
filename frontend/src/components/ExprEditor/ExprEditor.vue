<template>
  <div class="bb-risk-expr-editor text-sm w-full">
    <ConditionGroup :expr="expr" :root="true" @update="$emit('update')" />
  </div>
</template>

<script lang="ts" setup>
import { toRef } from "vue";
import type { ConditionGroupExpr, Factor, Operator } from "@/plugins/cel";
import ConditionGroup from "./ConditionGroup.vue";
import { provideExprEditorContext, type OptionConfig } from "./context";

const props = withDefaults(
  defineProps<{
    expr: ConditionGroupExpr;
    allowAdmin?: boolean;
    enableRawExpression?: boolean;
    factorList: Factor[];
    factorSupportDropdown?: Factor[];
    optionConfigMap?: Map<Factor, OptionConfig>;
    factorOperatorOverrideMap?: Map<Factor, Operator[]>;
  }>(),
  {
    allowAdmin: false,
    enableRawExpression: false,
    factorOperatorOverrideMap: undefined,
    factorSupportDropdown: () => [],
    optionConfigMap: () => new Map(),
  }
);

defineEmits<{
  (event: "update"): void;
}>();

provideExprEditorContext({
  allowAdmin: toRef(props, "allowAdmin"),
  enableRawExpression: toRef(props, "enableRawExpression"),
  factorList: toRef(props, "factorList"),
  factorSupportDropdown: toRef(props, "factorSupportDropdown"),
  optionConfigMap: toRef(props, "optionConfigMap"),
  factorOperatorOverrideMap: toRef(props, "factorOperatorOverrideMap"),
});
</script>

<style>
.bb-risk-expr-editor .n-base-selection {
  --n-padding-single: 0 22px 0 8px !important;
  --n-padding-multiple: 3px 24px 0 5px !important;
}
.bb-risk-expr-editor .n-base-selection .n-base-suffix {
  right: 5px !important;
}

.bb-risk-expr-editor .n-base-selection-tag-wrapper {
  padding-right: 2px;
}

.bb-risk-expr-editor .n-button {
  --n-padding: 0 6px 0 2px !important;
}
.bb-risk-expr-editor .n-button .n-button__icon {
  margin-right: 0px !important;
}
</style>
