<template>
  <NInput
    :value="value"
    :clearable="true"
    :placeholder="placeholder"
    style="width: 12rem"
    @update:value="$emit('update:value', $event)"
  >
    <template #prefix>
      <heroicons-outline:search class="h-4 w-4 text-control-placeholder" />
    </template>
  </NInput>
</template>

<script lang="ts" setup>
import { computed, useAttrs } from "vue";
import { useI18n } from "vue-i18n";
import { NInput } from "naive-ui";

withDefaults(
  defineProps<{
    value?: string;
  }>(),
  {
    value: "",
  }
);

defineEmits<{
  (event: "update:value", value: string): void;
}>();

const attrs = useAttrs();
const { t } = useI18n();

const placeholder = computed(() => {
  return (attrs.placeholder as string) ?? t("common.search");
});
</script>
