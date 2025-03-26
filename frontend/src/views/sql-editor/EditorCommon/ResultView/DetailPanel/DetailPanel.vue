<template>
  <DrawerContent
    :title="$t('common.detail')"
    class="w-[100vw-4rem] min-w-[24rem] max-w-[100vw-4rem] md:w-[33vw]"
  >
    <div
      class="h-full flex flex-col gap-y-2"
      :class="dark ? 'text-white' : 'text-main'"
    >
      <div class="flex items-center justify-between gap-x-4">
        <div class="flex items-center gap-x-2">
          <NTooltip :delay="500">
            <template #trigger>
              <NButton
                size="tiny"
                tag="div"
                :disabled="detail.row === 0"
                @click="move(-1)"
              >
                <template #icon>
                  <ChevronUpIcon class="w-4 h-4" />
                </template>
              </NButton>
            </template>
            <template #default>
              <div class="whitespace-nowrap">
                {{ $t("sql-editor.previous-row") }}
              </div>
            </template>
          </NTooltip>
          <NTooltip :delay="500">
            <template #trigger>
              <NButton
                size="tiny"
                tag="div"
                :disabled="detail.row === totalCount - 1"
                @click="move(1)"
              >
                <template #icon>
                  <ChevronDownIcon class="w-4 h-4" />
                </template>
              </NButton>
            </template>
            <template #default>
              <div class="whitespace-nowrap">
                {{ $t("sql-editor.next-row") }}
              </div>
            </template>
          </NTooltip>
          <div class="text-xs text-control-light flex items-center gap-x-1">
            <span>{{ detail.row + 1 }}</span>
            <span>/</span>
            <span>{{ totalCount }}</span>
            <span>{{ $t("sql-editor.rows", totalCount) }}</span>
          </div>
        </div>

        <div class="flex items-center gap-1">
          <NPopover v-if="guessedIsJSON">
            <template #trigger>
              <NButton
                size="small"
                style="--n-padding: 0 5px"
                :type="format ? 'primary' : 'default'"
                :secondary="format"
                @click="format = !format"
              >
                <template #icon>
                  <BracesIcon class="w-4 h-4" />
                </template>
              </NButton>
            </template>
            <template #default>
              {{ $t("sql-editor.format") }}
            </template>
          </NPopover>
          
          <!-- Binary data format selector -->
          <NPopover v-if="isBinaryData">
            <template #trigger>
              <NButton
                size="small"
                style="--n-padding: 0 5px"
                :type="binaryFormat !== serverFormat ? 'primary' : 'default'"
                :secondary="binaryFormat !== serverFormat"
              >
                <template #icon>
                  <Code2Icon class="w-4 h-4" />
                </template>
              </NButton>
            </template>
            <template #default>
              <div class="p-1">
                <NRadioGroup :value="binaryFormat" class="flex flex-col gap-2" @update:value="updateBinaryFormat">
                  <NRadio value="DEFAULT">
                    {{ $t("sql-editor.format-default") }}
                  </NRadio>
                  <NRadio value="BINARY">
                    {{ $t("sql-editor.binary-format") }}
                  </NRadio>
                  <NRadio value="HEX">
                    {{ $t("sql-editor.hex-format") }}
                  </NRadio>
                  <NRadio value="BOOLEAN" v-if="isSingleBitValue">
                    {{ $t("sql-editor.boolean-format") }}
                  </NRadio>
                  <NRadio value="TEXT">
                    {{ $t("sql-editor.text-format") }}
                  </NRadio>
                </NRadioGroup>
              </div>
            </template>
          </NPopover>
          
          <NButton v-if="!disallowCopyingData" size="small" @click="handleCopy">
            <template #icon>
              <ClipboardIcon class="w-4 h-4" />
            </template>
            {{ $t("common.copy") }}
          </NButton>
        </div>
      </div>
      <NScrollbar
        class="flex-1 overflow-hidden text-sm font-mono border p-2 relative"
        :content-class="contentClass"
        :x-scrollable="true"
        trigger="none"
      >
        <template v-if="guessedIsJSON && format">
          <div
            class="absolute right-2 top-2 flex justify-end items-center gap-1"
          >
            <NPopover>
              <template #trigger>
                <NButton
                  size="tiny"
                  style="--n-padding: 0 4px"
                  :type="wrap ? 'primary' : 'default'"
                  :secondary="wrap"
                  @click="wrap = !wrap"
                >
                  <template #icon>
                    <WrapTextIcon class="w-3 h-3" />
                  </template>
                </NButton>
              </template>
              <template #default>
                {{ $t("common.text-wrap") }}
              </template>
            </NPopover>
          </div>
          <PrettyJSON :content="content ?? ''" />
        </template>
        <template v-else>
          <template v-if="content && content.length > 0">
            {{ content }}
          </template>
          <br v-else style="min-width: 1rem; display: inline-flex" />
        </template>
      </NScrollbar>
    </div>
  </DrawerContent>
</template>

<script setup lang="ts">
import { onKeyStroke, useClipboard, useLocalStorage } from "@vueuse/core";
import {
  ChevronDownIcon,
  ChevronUpIcon,
  ClipboardIcon,
  BracesIcon,
  WrapTextIcon,
  Code as Code2Icon,
} from "lucide-vue-next";
import { NButton, NPopover, NScrollbar, NTooltip, NRadioGroup, NRadio } from "naive-ui";
import { computed, nextTick, ref } from "vue";
import { useI18n } from "vue-i18n";
import { DrawerContent } from "@/components/v2";
import { pushNotification } from "@/store";
import type { RowValue } from "@/types/proto/v1/sql_service";
import { extractSQLRowValuePlain } from "@/utils";
import { useSQLResultViewContext } from "../context";
import PrettyJSON from "./PrettyJSON.vue";

const { t } = useI18n();
const { dark, detail, disallowCopyingData } = useSQLResultViewContext();

const format = useLocalStorage<boolean>(
  "bb.sql-editor.detail-panel.format",
  false
);
const wrap = useLocalStorage<boolean>(
  "bb.sql-editor.detail-panel.line-wrap",
  true
);

// Get the current value being displayed first
const rawValue = computed(() => {
  const { row, col, table } = detail.value;
  if (!table) return undefined;

  return table
    .getPrePaginationRowModel()
    .rows[row]?.getVisibleCells()
    [col]?.getValue<RowValue>();
});

// Determine binary format based on column type and content
const getServerFormat = (): string => {
  if (!rawValue.value?.bytesValue) return "HEX";
  
  // Get column information
  const { col, table } = detail.value;
  if (!table) return "HEX";
  
  // Get the column type from context's columnTypeNames
  let columnType = '';
  
  // Get context and safely access columnTypeNames
  const context = useSQLResultViewContext();
  const columnTypeNames = context.columnTypeNames?.value;
  
  // Use column type names from context
  if (columnTypeNames && col < columnTypeNames.length) {
    columnType = columnTypeNames[col].toLowerCase();
  }
  
  // Default format based on column type
  let defaultFormat = "HEX";
  
  // Detect BIT column types (bit, varbit, bit varying) - for binary format display
  const isBitColumn = (
    // Generic bit types
    columnType === 'bit' ||
    columnType.startsWith('bit(') ||
    (columnType.includes('bit') && !columnType.includes('binary')) ||
    
    // PostgreSQL bit types
    columnType === 'varbit' ||
    columnType === 'bit varying'
  );
    
  // Detect BINARY column types (binary, varbinary, bytea, blob, etc) - for hex format display
  const isBinaryColumn = (
    // Generic binary types
    columnType === 'binary' ||
    columnType.includes('binary') || 
    
    // MySQL/MariaDB binary types
    columnType.startsWith('binary(') ||
    columnType.startsWith('varbinary') ||
    columnType.includes('blob') ||
    columnType === 'longblob' ||
    columnType === 'mediumblob' ||
    columnType === 'tinyblob' ||
    
    // PostgreSQL binary type
    columnType === 'bytea' ||
    
    // SQL Server binary types
    columnType === 'image' ||
    columnType === 'varbinary(max)' ||
    
    // Oracle binary types
    columnType === 'raw' ||
    columnType === 'long raw'
  );
  
  // BIT columns default to binary format
  if (isBitColumn) {
    defaultFormat = "BINARY";
  }
  
  // BINARY/VARBINARY/BLOB columns default to HEX format
  if (isBinaryColumn) {
    defaultFormat = "HEX";
  }
  
  const byteArray = Array.from(rawValue.value.bytesValue);
  
  // For single byte values (could be boolean)
  if (byteArray.length === 1 && (byteArray[0] === 0 || byteArray[0] === 1)) {
    return "BOOLEAN";
  }
  
  // Check if it's readable text
  const isReadableText = byteArray.every(byte => byte >= 32 && byte <= 126);
  if (isReadableText) {
    return "TEXT";
  }
  
  // Return default format based on column type
  return defaultFormat;
};

// Current display format (reactive to server changes)
const serverFormat = computed(() => getServerFormat());

// Create a ref for temporary format override
const formatOverride = ref<string | null>(null);

// The actual format to display
const binaryFormat = computed(() => {
  if (formatOverride.value === null) {
    return "DEFAULT"; // No override, show as DEFAULT in radio group
  }
  return formatOverride.value;
});

// Check if the current value is binary data (using bytesValue)
const isBinaryData = computed(() => {
  if (!rawValue.value) return false;
  return !!rawValue.value.bytesValue;
});

// Check if it's a single bit value (for boolean display)
const isSingleBitValue = computed(() => {
  if (!rawValue.value?.bytesValue) return false;
  return rawValue.value.bytesValue.length === 1;
});

// Format the binary value based on selected format
const formattedBinaryValue = computed(() => {
  if (!rawValue.value?.bytesValue) return rawValue.value;
  
  // Deep clone the value to avoid mutating the original
  const newValue = { ...rawValue.value };
  
  // Get the actual format to use (override or server default)
  const actualFormat = formatOverride.value === null ? serverFormat.value : formatOverride.value;
  
  // Skip formatting if using default format
  if (actualFormat === "DEFAULT") {
    return newValue;
  }
  
  // Ensure bytesValue exists before converting to array
  const byteArray = newValue.bytesValue ? Array.from(newValue.bytesValue) : [];
  
  // Format based on selected format
  switch (actualFormat) {
    case "BINARY":
      // Format as binary string without spaces
      newValue.stringValue = byteArray
        .map(byte => byte.toString(2).padStart(8, "0"))
        .join("");
      break;
    case "HEX":
      // Format as hex string
      newValue.stringValue = "0x" + byteArray
        .map(byte => byte.toString(16).toUpperCase().padStart(2, "0"))
        .join("");
      break;
    case "TEXT":
      // Format as text
      try {
        newValue.stringValue = new TextDecoder().decode(new Uint8Array(byteArray));
      } catch {
        // Fallback to binary if text decoding fails
        newValue.stringValue = byteArray
          .map(byte => byte.toString(2).padStart(8, "0"))
          .join("");
      }
      break;
    case "BOOLEAN":
      // Only for single-byte values
      if (byteArray.length === 1) {
        newValue.stringValue = byteArray[0] === 1 ? "true" : "false";
      }
      break;
  }
  
  return newValue;
});

const content = computed(() => {
  if (!rawValue.value) return undefined;
  
  // If it's binary data, use the formatted version
  if (isBinaryData.value) {
    return String(extractSQLRowValuePlain(formattedBinaryValue.value));
  }
  
  // Otherwise use the raw value
  return String(extractSQLRowValuePlain(rawValue.value));
});

const guessedIsJSON = computed(() => {
  if (!content.value || content.value.length === 0) return false;
  const maybeJSON = content.value.trim();
  return (
    (maybeJSON.startsWith("{") && maybeJSON.endsWith("}")) ||
    (maybeJSON.startsWith("[") && maybeJSON.endsWith("]"))
  );
});

const totalCount = computed(() => {
  const { table } = detail.value;
  if (!table) return 0;
  return table.getPrePaginationRowModel().rows.length;
});

const contentClass = computed(() => {
  const classes: string[] = [];

  if (disallowCopyingData.value) {
    classes.push("select-none");
  }
  if (guessedIsJSON.value && format.value && !wrap.value) {
    classes.push("whitespace-pre");
  } else {
    classes.push("whitespace-pre-wrap");
  }
  return classes.join(" ");
});

const { copy, copied } = useClipboard({
  source: computed(() => {
    const raw = content.value ?? "";
    
    // For JSON content
    if (guessedIsJSON.value && format.value) {
      try {
        const obj = JSON.parse(raw);
        return JSON.stringify(obj, null, "  ");
      } catch {
        console.warn(
          "[DetailPanel]",
          "failed to parse and format (maybe) JSON value"
        );
        return raw;
      }
    }
    
    // For binary data, copy according to the selected format
    if (isBinaryData.value) {
      return raw;
    }
    
    return raw;
  }),
  legacy: true,
});
const handleCopy = () => {
  copy().then(() => {
    if (copied.value) {
      pushNotification({
        module: "bytebase",
        style: "INFO",
        title: t("common.copied"),
      });
    }
  });
};

const move = (offset: number) => {
  const target = detail.value.row + offset;
  if (target < 0 || target >= totalCount.value) return;
  detail.value.row = target;
};

onKeyStroke("ArrowUp", (e) => {
  e.preventDefault();
  e.stopPropagation();
  move(-1);
});
onKeyStroke("ArrowDown", (e) => {
  e.preventDefault();
  e.stopPropagation();
  move(1);
});

// Method to handle binary format updates and ensure re-rendering
const updateBinaryFormat = (value: string) => {
  // If DEFAULT is selected, remove the override
  if (value === "DEFAULT") {
    formatOverride.value = null;
  } else {
    // Otherwise set the override
    formatOverride.value = value;
  }
  
  // Force render
  nextTick();
};
</script>
