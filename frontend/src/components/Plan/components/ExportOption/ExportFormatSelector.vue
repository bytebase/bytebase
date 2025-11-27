<template>
  <NRadioGroup v-if="editable" :value="format" @update:value="handleUpdate">
    <NRadio
      v-for="formatItem in availableExportFormats"
      :key="formatItem"
      :value="formatItem"
      :label="ExportFormat[formatItem]"
    />
  </NRadioGroup>
  <template v-else>
    <span class="text-sm font-medium leading-6">{{
      ExportFormat[format]
    }}</span>
  </template>
</template>

<script setup lang="ts">
import { NRadio, NRadioGroup } from "naive-ui";
import { computed, onMounted } from "vue";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";

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
