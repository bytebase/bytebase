<template>
  <NRadioGroup
    :value="value"
    @update:value="$emit('update:value', $event as boolean)"
  >
    <NRadio :value="false">{{ getLabel(false) }}</NRadio>
    <NRadio :value="true">{{ getLabel(true) }}</NRadio>
  </NRadioGroup>
</template>

<script setup lang="ts">
import { NRadio, NRadioGroup } from "naive-ui";
import { useI18n } from "vue-i18n";

const props = defineProps<{
  value: boolean;
  defaults: boolean;
  readonly: boolean;
}>();

defineEmits<{
  (event: "update:value", on: boolean): void;
}>();

const { t } = useI18n();

const getLabel = (value: boolean) => {
  const label = value ? "true" : "false";
  if (value === props.defaults) {
    return `(${t("common.default")}) ${label}`;
  }
  return label;
};
</script>
