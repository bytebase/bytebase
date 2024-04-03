<template>
  <NRadioGroup v-if="editable" :value="format" @update:value="handleUpdate">
    <NRadio
      v-for="formatItem in availableExportFormats"
      :key="formatItem"
      :value="formatItem"
      :label="exportFormatToJSON(formatItem)"
    />
  </NRadioGroup>
  <template v-else>
    <span class="text-base">{{ exportFormatToJSON(format) }}</span>
  </template>
</template>

<script setup lang="ts">
import { NRadioGroup, NRadio } from "naive-ui";
import { computed } from "vue";
import { onMounted } from "vue";
import { ExportFormat, exportFormatToJSON } from "@/types/proto/v1/common";

const props = defineProps<{
  format: ExportFormat;
  editable?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:format", value: ExportFormat): void;
}>();

const availableExportFormats = computed(() => [
  ExportFormat.JSON,
  ExportFormat.CSV,
  ExportFormat.SQL,
  ExportFormat.XLSX,
]);

const handleUpdate = (value: ExportFormat) => {
  emit("update:format", value);
};

onMounted(() => {
  if (!availableExportFormats.value.includes(props.format)) {
    handleUpdate(availableExportFormats.value[0]);
  }
});
</script>
