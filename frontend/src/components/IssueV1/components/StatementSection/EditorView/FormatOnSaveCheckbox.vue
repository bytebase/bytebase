<template>
  <NCheckbox
    v-if="allowFormatOnSave"
    :checked="value"
    size="small"
    @update:checked="$emit('update:value', $event)"
  >
    {{ $t("issue.format-on-save") }}
  </NCheckbox>
</template>

<script setup lang="ts">
import { NCheckbox } from "naive-ui";
import { computed } from "vue";
import { Language } from "@/types";

const props = defineProps<{
  language: Language;
  value: boolean;
}>();

defineEmits<{
  (event: "update:value", on: boolean): void;
}>();

const allowFormatOnSave = computed(() => props.language === "sql");
</script>
