<template>
  <NPerformantEllipsis v-if="showText" :line-clamp="1">
    <span v-if="columnClassification">
      {{ columnClassification.title }}
    </span>
    <span v-else class="text-control-placeholder italic">
      {{ placeholder }}
    </span>
  </NPerformantEllipsis>
  <span v-if="level" :class="['ml-1 px-1 py-0.5 rounded text-xs', levelColor]">
    {{ level.title }}
  </span>
</template>

<script lang="ts" setup>
import { NPerformantEllipsis } from "naive-ui";
import { computed } from "vue";
import type { DataClassificationSetting_DataClassificationConfig } from "@/types/proto-es/v1/setting_service_pb";

const props = withDefaults(
  defineProps<{
    showText?: boolean;
    classification?: string;
    classificationConfig?: DataClassificationSetting_DataClassificationConfig;
    placeholder?: string;
  }>(),
  {
    showText: true,
    classification: undefined,
    classificationConfig: undefined,
    placeholder: "N/A",
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
