<!-- Button to change binary data format for an entire column -->
<template>
  <div>
    <NPopover trigger="click" placement="bottom">
      <template #trigger>
        <NButton
          size="tiny"
          circle
          class="ml-1 dark:!bg-dark-bg"
          @click.stop
          :type="hasColumnOverride ? 'primary' : 'default'"
          :secondary="hasColumnOverride"
        >
          <template #icon>
            <IconCode class="w-3 h-3" />
          </template>
        </NButton>
      </template>
      <template #default>
        <div class="p-2 w-52">
          <div class="text-xs font-semibold mb-2">{{ $t("sql-editor.column-display-format") }}</div>
          <NRadioGroup
            :value="currentColumnFormat"
            class="flex flex-col gap-2"
            @update:value="updateColumnFormat"
          >
            <NRadio value="DEFAULT">
              {{ $t("sql-editor.format-default") }}
            </NRadio>
            <NRadio value="BINARY">
              {{ $t("sql-editor.binary-format") }}
            </NRadio>
            <NRadio value="HEX">
              {{ $t("sql-editor.hex-format") }}
            </NRadio>
            <NRadio value="TEXT">
              {{ $t("sql-editor.text-format") }}
            </NRadio>
            <NRadio value="BOOLEAN" v-if="hasSingleBitValues">
              {{ $t("sql-editor.boolean-format") }}
            </NRadio>
          </NRadioGroup>
        </div>
      </template>
    </NPopover>
  </div>
</template>

<script setup lang="ts">
import { Code as IconCode } from "lucide-vue-next";
import { NButton, NPopover, NRadioGroup, NRadio } from "naive-ui";
import { computed } from "vue";

// Event interface
const emit = defineEmits<{
  (e: "update:format", format: string | null): void;
}>();

// Props
const props = defineProps<{
  // Current column index
  columnIndex: number;
  // Current format override for the column
  columnFormat: string | null;
  // Auto-detected server format
  serverFormat: string | null;
  // Whether this column contains any single-bit values (for boolean option)
  hasSingleBitValues: boolean;
}>();

// Track if there's currently a format override for the column
const hasColumnOverride = computed(() => {
  return props.columnFormat !== null;
});

// The currently selected format for the radio group
const currentColumnFormat = computed(() => {
  return props.columnFormat || "DEFAULT";
});

// Update the column format
const updateColumnFormat = (format: string) => {
  if (format === "DEFAULT") {
    // Set to null to use server default
    emit("update:format", null);
  } else {
    // Set to specific format
    emit("update:format", format);
  }
};
</script>