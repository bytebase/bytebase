<template>
  <NInput
    :value="value"
    :placeholder="placeholder"
    :disabled="!allowAdmin"
    size="small"
    style="min-width: 5rem; width: auto; overflow-x: hidden"
    @update:value="$emit('update:value', $event)"
  />
</template>

<script lang="ts" setup>
import { NInput } from "naive-ui";
import { t } from "@/plugins/i18n";
import { useExprEditorContext } from "../context";

withDefaults(
  defineProps<{
    value: string;
    placeholder?: string;
  }>(),
  {
    placeholder: () => t("cel.condition.input-value"),
  }
);

defineEmits<{
  (event: "update:value", value: string): void;
}>();

const context = useExprEditorContext();
const { allowAdmin } = context;
</script>
