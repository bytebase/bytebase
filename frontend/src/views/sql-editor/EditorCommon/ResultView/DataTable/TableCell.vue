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
      <!-- Format indicator for ByteData - only show when no column format is set -->
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
import { RowValue_ByteData_DisplayFormat } from "@/types/proto/v1/sql_service";
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

// Get the server-provided format
const getServerFormat = (): string => {
  if (!props.value.byteDataValue) return "BINARY";
  
  // If it has a display format already specified, use that
  if (props.value.byteDataValue.displayFormat) {
    return props.value.byteDataValue.displayFormat;
  }
  
  // If no format is specified, determine intelligently based on data
  const byteArray = Array.from(props.value.byteDataValue.value);
  
  // For single byte values (could be boolean)
  if (byteArray.length === 1 && (byteArray[0] === 0 || byteArray[0] === 1)) {
    return "BOOLEAN";
  }
  
  // Check if it's readable text
  const isReadableText = byteArray.every(byte => byte >= 32 && byte <= 126);
  if (isReadableText) {
    return "TEXT";
  }
  
  // Default to HEX for most binary data as it's more compact than binary
  return "HEX";
};

// Current display format (reactive to server changes)
const serverFormat = computed(() => getServerFormat());

// Create a ref for temporary format override
const formatOverride = ref<string | null>(null);

// The actual format to display - use column override, cell override, or server format
const displayFormat = computed(() => {
  // Column-level override takes precedence (if provided)
  if (props.columnFormatOverride && props.columnFormatOverride !== "DEFAULT") {
    return props.columnFormatOverride;
  }
  
  // Cell-level override is next
  if (formatOverride.value && formatOverride.value !== "DEFAULT") {
    return formatOverride.value;
  }
  
  // Fallback to server format
  return serverFormat.value;
});

const allowSelect = computed(() => {
  return props.allowSelect && !selectionDisabled.value;
});

// Check if the value is ByteData
const hasByteData = computed(() => {
  return !!props.value.byteDataValue;
});

// Check if it's a single bit ByteData (for boolean display)
const isSingleBitValue = computed(() => {
  if (props.value.byteDataValue) {
    return props.value.byteDataValue.value.length === 1;
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

// If it's ByteData, create a formatted value with the chosen format
const formattedValue = computed(() => {
  if (!hasByteData.value) return props.value;
  
  // Create a clone to avoid modifying the original
  const formatted = { ...props.value };
  
  if (formatted.byteDataValue) {
    // Create a new ByteData object with the same value but updated format
    formatted.byteDataValue = {
      value: formatted.byteDataValue.value,
      displayFormat: displayFormat.value as RowValue_ByteData_DisplayFormat
    };
  }
  
  return formatted;
});

const html = computed(() => {
  let value;
  
  // Use the formatted value if it's ByteData
  if (hasByteData.value) {
    value = extractSQLRowValuePlain(formattedValue.value);
  } else {
    value = extractSQLRowValuePlain(props.value);
  }
  
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

// Handle right-click context menu for ByteData
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
  nextTick(() => {
    // This empty callback just ensures the component re-renders
  });
};
</script>
