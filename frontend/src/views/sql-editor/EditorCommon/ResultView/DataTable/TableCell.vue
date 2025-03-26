<!-- eslint-disable vue/no-v-html -->
<template>
  <div
    class="relative px-2 py-1"
    :class="classes"
    @click="handleClick"
    @dblclick="showDetail"
    @contextmenu.prevent="handleContextMenu"
    ref="cellRef"
  >
    <div
      ref="wrapperRef"
      class="whitespace-nowrap font-mono text-start line-clamp-1"
      v-html="html"
    ></div>
    <div class="absolute right-1 top-1/2 translate-y-[-45%] flex items-center gap-1">
      <!-- Format button for binary data -->
      <NButton
        v-if="hasByteData && !props.columnFormatOverride"
        size="tiny"
        circle
        class="dark:!bg-dark-bg"
        @click.stop="showFormatDropdown"
      >
        <template #icon>
          <IconCode class="w-3 h-3" />
        </template>
      </NButton>
      
      <!-- Expand button for long content -->
      <NButton
        v-if="clickable"
        size="tiny"
        circle
        class="dark:!bg-dark-bg"
        @click.stop="showDetail"
      >
        <template #icon>
          <heroicons:arrows-pointing-out class="w-3 h-3" />
        </template>
      </NButton>
    </div>
    
    <!-- Format dropdown menu -->
    <NDropdown
      v-if="hasByteData"
      :trigger="'manual'"
      :show="showFormatMenu"
      :options="formatOptions"
      :x="dropdownX"
      :y="dropdownY"
      @select="handleFormatSelect"
      @clickoutside="showFormatMenu = false"
    />
  </div>
</template>

<script setup lang="ts">
import { type Table } from "@tanstack/vue-table";
import { useResizeObserver } from "@vueuse/core";
import { escape } from "lodash-es";
import { Code as IconCode } from "lucide-vue-next";
import { NButton, NDropdown } from "naive-ui";
import { twMerge } from "tailwind-merge";
import { computed, nextTick, ref } from "vue";
import { useConnectionOfCurrentSQLEditorTab } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import type { QueryRow, RowValue } from "@/types/proto/v1/sql_service";
import { extractSQLRowValuePlain, getHighlightHTMLByRegExp } from "@/utils";
import { useSQLResultViewContext } from "../context";
import { useSelectionContext } from "./common/selection-logic";
import { useI18n } from "vue-i18n";

const props = defineProps<{
  table: Table<QueryRow>;
  value: RowValue;
  setIndex: number;
  rowIndex: number;
  colIndex: number;
  allowSelect?: boolean;
  columnFormatOverride?: string | null;
  columnType?: string; // Column type from QueryResult
}>();

const { t } = useI18n();
const { detail, keyword } = useSQLResultViewContext();

const {
  state: selectionState,
  disabled: selectionDisabled,
  selectCell,
  selectRow,
} = useSelectionContext();
const wrapperRef = ref<HTMLDivElement>();
const cellRef = ref<HTMLDivElement>();
const truncated = ref(false);

// Dropdown menu state
const showFormatMenu = ref(false);
const dropdownX = ref(0);
const dropdownY = ref(0);

// Create a ref for temporary format override
const formatOverride = ref<string | null>(null);

const allowSelect = computed(() => {
  return props.allowSelect && !selectionDisabled.value;
});

// Check if the value is binary data
const hasByteData = computed(() => {
  return !!props.value.bytesValue;
});

// Check if it's a single bit value (for boolean display)
const isSingleBitValue = computed(() => {
  if (props.value.bytesValue) {
    return props.value.bytesValue.length === 1;
  }
  return false;
});

// Format options for the dropdown
const formatOptions = computed(() => {
  const currentFormat = formatOverride.value === null ? "DEFAULT" : formatOverride.value;
  
  const options = [
    {
      label: t("sql-editor.format-default"),
      key: "DEFAULT",
      disabled: false,
    },
    {
      label: t("sql-editor.binary-format"),
      key: "BINARY",
      disabled: false,
    },
    {
      label: t("sql-editor.hex-format"),
      key: "HEX",
      disabled: false,
    },
    {
      label: t("sql-editor.text-format"),
      key: "TEXT",
      disabled: false,
    },
  ];
  
  // Only show boolean option for single-byte values
  if (isSingleBitValue.value) {
    options.splice(3, 0, {
      label: t("sql-editor.boolean-format"),
      key: "BOOLEAN",
      disabled: false,
    });
  }
  
  // Add a checkmark to the currently selected option
  options.forEach(option => {
    if (option.key === currentFormat) {
      option.label = "✓ " + option.label;
    }
  });
  
  return options;
});

useResizeObserver(wrapperRef, (entries) => {
  const div = entries[0].target as HTMLDivElement;
  const contentWidth = div.scrollWidth;
  const visibleWidth = div.offsetWidth;
  if (contentWidth > visibleWidth) {
    truncated.value = true;
  } else {
    truncated.value = false;
  }
});
const { database } = useConnectionOfCurrentSQLEditorTab();

const clickable = computed(() => {
  if (truncated.value) return true;
  if (database.value.instanceResource.engine === Engine.MONGODB) {
    // A cheap way to check JSON string without paying the parsing cost.
    const maybeJSON = String(props.value).trim();
    return (
      (maybeJSON.startsWith("{") && maybeJSON.endsWith("}")) ||
      (maybeJSON.startsWith("[") && maybeJSON.endsWith("]"))
    );
  }
  return false;
});

const classes = computed(() => {
  const classes: string[] = [];
  if (allowSelect.value) {
    classes.push("cursor-pointer");
    classes.push("hover:bg-white/20 dark:hover:bg-black/5");
    if (props.colIndex === 0) {
      classes.push("pl-3");
    }
    if (
      selectionState.value.columns.length === 1 &&
      selectionState.value.rows.length === 1
    ) {
      if (
        selectionState.value.columns[0] === props.colIndex &&
        selectionState.value.rows[0] === props.rowIndex
      ) {
        classes.push("!bg-accent/10 dark:!bg-accent/40");
      }
    } else if (
      selectionState.value.columns.includes(props.colIndex) ||
      selectionState.value.rows.includes(props.rowIndex)
    ) {
      classes.push("!bg-accent/10 dark:!bg-accent/40");
    } else {
    }
  } else {
    classes.push("select-none");
  }
  return twMerge(classes);
});

// Format the binary value based on selected format
const formattedBinaryValue = computed(() => {
  if (!props.value?.bytesValue) return props.value;
  
  // Deep clone the value to avoid mutating the original
  const newValue = { ...props.value };
  
  // Determine the format to use - column override, cell override, or auto-detected format
  let actualFormat = null;
  
  // If column has an override, use that
  if (props.columnFormatOverride !== null && props.columnFormatOverride !== undefined) {
    actualFormat = props.columnFormatOverride;
  } 
  // Otherwise use the cell's override if set
  else if (formatOverride.value !== null) {
    actualFormat = formatOverride.value;
  }
  // Otherwise use auto-detected format
  else {
    // Use column type directly from props if available
    const columnType = props.columnType?.toLowerCase() || '';
    
    // Default format based on column type
    let defaultFormat = "BINARY";
    
    // Detect BIT column types - for binary format display (0s and 1s)
    const isBitColumn = (
      columnType === 'bit' ||
      columnType.startsWith('bit(') ||
      (columnType.includes('bit') && !columnType.includes('binary')) ||
      columnType === 'varbit' ||
      columnType === 'bit varying'
    );
    
    // Detect BINARY column types - for hex format display (0x...)
    const isBinaryColumn = (
      columnType === 'binary' ||
      columnType.includes('binary') ||
      columnType.startsWith('binary(') ||
      columnType.startsWith('varbinary') ||
      columnType.includes('blob') ||
      columnType === 'bytea'
    );
    
    // Set default format based on column type
    if (isBitColumn) {
      defaultFormat = "BINARY";
    } else if (isBinaryColumn) {
      defaultFormat = "HEX";
    }
    
    // Now also consider content-based auto-detection
    const byteArray = newValue.bytesValue ? Array.from(newValue.bytesValue) : [];
    
    // For single bit values (could be boolean)
    if (byteArray.length === 1 && (byteArray[0] === 0 || byteArray[0] === 1)) {
      actualFormat = "BOOLEAN";
    }
    // Check if it's readable text
    else if (byteArray.every(byte => byte >= 32 && byte <= 126)) {
      actualFormat = "TEXT";
    }
    // Default to format based on column type
    else {
      actualFormat = defaultFormat;
    }
  }
  
  // Skip formatting for DEFAULT (auto) format
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
      } else {
        // Fall back to BINARY for non-single-bit values
        newValue.stringValue = byteArray
          .map(byte => byte.toString(2).padStart(8, "0"))
          .join("");
      }
      break;
  }
  
  return newValue;
});

const html = computed(() => {
  // If it's binary data, use the formatted version
  const valueToRender = hasByteData.value ? formattedBinaryValue.value : props.value;
  
  // Extract the display value
  const value = extractSQLRowValuePlain(valueToRender);
  
  if (value === undefined) {
    return `<span class="text-gray-400 italic">UNSET</span>`;
  }
  if (value === null) {
    return `<span class="text-gray-400 italic">NULL</span>`;
  }
  const str = String(value);
  if (str.length === 0) {
    return `<br style="min-width: 1rem; display: inline-flex;" />`;
  }

  const kw = keyword.value.trim();
  if (!kw) {
    return escape(str);
  }

  return getHighlightHTMLByRegExp(
    escape(str),
    escape(kw),
    false /* !caseSensitive */
  );
});

const handleClick = (e: MouseEvent) => {
  if (!allowSelect.value) {
    return;
  }
  // If the user is holding the ctrl/cmd key, select the row.
  if (e.ctrlKey || e.metaKey) {
    selectRow(props.rowIndex);
  } else {
    selectCell(props.rowIndex, props.colIndex);
  }
  e.stopPropagation();
};

const showDetail = () => {
  detail.value = {
    show: true,
    set: props.setIndex,
    row: props.rowIndex,
    col: props.colIndex,
    table: props.table,
  };
};

// Display the format dropdown menu
const showFormatDropdown = (e: MouseEvent) => {
  if (!cellRef.value) return;
  
  // Position the dropdown relative to the click position
  dropdownX.value = e.clientX;
  dropdownY.value = e.clientY;
  showFormatMenu.value = true;
};

// Handle right-click context menu for binary data
const handleContextMenu = (e: MouseEvent) => {
  if (hasByteData.value) {
    showFormatDropdown(e);
  }
};

// Handle format selection from dropdown
const handleFormatSelect = (key: string) => {
  // If DEFAULT is selected, remove the override
  if (key === "DEFAULT") {
    formatOverride.value = null;
  } else {
    // Otherwise set the override
    formatOverride.value = key;
  }
  
  showFormatMenu.value = false;
  
  // Force recomputation of html
  nextTick();
};
</script>
