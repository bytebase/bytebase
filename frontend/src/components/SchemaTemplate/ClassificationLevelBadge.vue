<template>
  {{ showText ? columnClassification?.title ?? placeholder : "" }}
  <span v-if="level" :class="['px-1 py-0.5 rounded text-xs', levelColor]">
    {{ level.title }}
  </span>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { DataClassificationSetting_DataClassificationConfig } from "@/types/proto/v1/setting_service";

const props = withDefaults(
  defineProps<{
    showText?: boolean;
    classification?: string;
    classificationConfig?: DataClassificationSetting_DataClassificationConfig;
    placeholder?: string;
  }>(),
  {
    showText: true,
  }
);

const bgColorList = [
  "bg-green-200",
  "bg-yellow-200",
  "bg-orange-300",
  "bg-amber-500",
  "bg-red-500",
];

const columnClassification = computed(() => {
  if (!props.classification || !props.classificationConfig) {
    return;
  }
  return props.classificationConfig.classification[props.classification];
});

const levelColor = computed(() => {
  const index = (props.classificationConfig?.levels ?? []).findIndex(
    (level) => level.id === columnClassification.value?.levelId
  );
  return bgColorList[index] ?? "bg-gray-200";
});

const level = computed(() => {
  return (props.classificationConfig?.levels ?? []).find(
    (level) => level.id === columnClassification.value?.levelId
  );
});
</script>
