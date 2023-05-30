<template>
  <NSelect
    :value="value"
    :filterable="true"
    :multiple="true"
    :tag="true"
    :show-arrow="false"
    :show="false"
    :consistent-menu-width="false"
    :placeholder="
      $t('custom-approval.security-rule.condition.input-value-press-enter')
    "
    :disabled="!allowAdmin"
    max-tag-count="responsive"
    size="small"
    style="min-width: 16rem; width: auto; overflow-x: hidden"
    @update:value="$emit('update:value', $event)"
  />
</template>

<script lang="ts" setup>
import { NSelect } from "naive-ui";
import { useExprEditorContext } from "../context";

defineProps<{
  value: string[];
}>();

defineEmits<{
  (event: "update:value", value: string[]): void;
}>();

const context = useExprEditorContext();
const { allowAdmin } = context;
</script>
