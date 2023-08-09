<template>
  <div class="bb-expr-editor text-sm w-full">
    <ConditionGroup :expr="expr" :root="true" @update="$emit('update')" />
  </div>
</template>

<script lang="ts" setup>
import { toRef } from "vue";
import type { ConditionGroupExpr } from "@/plugins/cel";
import ConditionGroup from "./ConditionGroup.vue";
import { ResourceType, provideExprEditorContext } from "./context";

const props = withDefaults(
  defineProps<{
    expr: ConditionGroupExpr;
    resourceType: ResourceType;
    allowAdmin?: boolean;
  }>(),
  {
    resourceType: "DATABASE_GROUP",
    allowAdmin: false,
  }
);

defineEmits<{
  (event: "update"): void;
}>();

provideExprEditorContext({
  allowAdmin: toRef(props, "allowAdmin"),
  resourceType: toRef(props, "resourceType"),
});
</script>

<style>
.bb-expr-editor .n-base-selection {
  --n-padding-single: 0 22px 0 8px !important;
  --n-padding-multiple: 3px 24px 0 5px !important;
}
.bb-expr-editor .n-base-selection .n-base-suffix {
  right: 5px !important;
}

.bb-expr-editor .n-base-selection-tag-wrapper {
  padding-right: 2px;
}

.bb-expr-editor .n-button {
  --n-padding: 0 6px 0 2px !important;
}
.bb-expr-editor .n-button .n-button__icon {
  margin-right: 0px !important;
}
</style>
