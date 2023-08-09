<template>
  <div class="bb-risk-expr-editor text-sm w-full">
    <ConditionGroup :expr="expr" :root="true" @update="$emit('update')" />
  </div>
</template>

<script lang="ts" setup>
import { toRef } from "vue";
import type { ConditionGroupExpr } from "@/plugins/cel";
import { Risk_Source } from "@/types/proto/v1/risk_service";
import ConditionGroup from "./ConditionGroup.vue";
import { provideExprEditorContext } from "./context";

const props = withDefaults(
  defineProps<{
    expr: ConditionGroupExpr;
    allowAdmin?: boolean;
    allowHighLevelFactors?: boolean;
    riskSource?: Risk_Source;
  }>(),
  {
    allowAdmin: false,
    allowHighLevelFactors: false,
    riskSource: undefined,
  }
);

defineEmits<{
  (event: "update"): void;
}>();

provideExprEditorContext({
  allowAdmin: toRef(props, "allowAdmin"),
  allowHighLevelFactors: toRef(props, "allowHighLevelFactors"),
  riskSource: toRef(props, "riskSource"),
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
