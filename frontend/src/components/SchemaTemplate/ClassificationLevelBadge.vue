<template>
  <span
    v-if="level"
    :class="[
      'px-2 py-1 rounded text-xs',
      bgColorList[Number(level.id)] ?? 'bg-gray-200',
    ]"
  >
    {{ level.title }}
  </span>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { DataClassificationSetting_DataClassificationConfig } from "@/types/proto/v1/setting_service";

const props = defineProps<{
  levelId?: string;
  classificationConfig: DataClassificationSetting_DataClassificationConfig;
}>();

const bgColorList = [
  "bg-green-200",
  "bg-yellow-200",
  "bg-orange-300",
  "bg-amber-500",
  "bg-red-500",
];

const level = computed(() => {
  return (props.classificationConfig.levels ?? []).find(
    (level) => level.id === props.levelId
  );
});
</script>
