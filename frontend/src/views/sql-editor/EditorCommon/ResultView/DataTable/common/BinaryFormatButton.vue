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
          <div class="text-xs font-semibold mb-2">Column Display Format</div>
          <NRadioGroup
            :value="currentColumnFormat"
            class="flex flex-col gap-2"
            @update:value="updateColumnFormat"
          >
            <NRadio value="DEFAULT">
              Default
            </NRadio>
            <NRadio value="BINARY">
              Binary (0s and 1s)
            </NRadio>
            <NRadio value="HEX">
              Hexadecimal (0x...)
            </NRadio>
            <NRadio value="TEXT">
              Text (UTF-8)
            </NRadio>
            <NRadio value="BOOLEAN" v-if="hasSingleBitValues">
              Boolean (true/false)
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
import { computed, ref } from "vue";
import { RowValue_ByteData_DisplayFormat } from "@/types/proto/v1/sql_service";

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
  // Server-provided default format
  serverFormat: string | null;
  // Whether this column contains any single-bit values
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