<template>
  <NInput
    ref="inputRef"
    :value="value"
    :clearable="!!value"
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
import { NInput } from "naive-ui";
import { computed, onMounted, ref, useAttrs } from "vue";
import { useI18n } from "vue-i18n";

const props = withDefaults(
  defineProps<{
    value?: string;
    autofocus?: boolean;
  }>(),
  {
    value: "",
    autofocus: false,
  }
);

defineEmits<{
  (event: "update:value", value: string): void;
}>();

const inputRef = ref<InstanceType<typeof NInput>>();
const attrs = useAttrs();
const { t } = useI18n();

const placeholder = computed(() => {
  return (attrs.placeholder as string) ?? t("common.search");
});

onMounted(() => {
  if (props.autofocus) {
    inputRef.value?.inputElRef?.focus();
  }
});
</script>
