<template>
  <DrawerContent
    v-if="detail"
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
          <BinaryFormatButton
            v-if="isBinaryData"
            :format="binaryFormat"
            @update:format="
              (format: BinaryFormat) =>
                setBinaryFormat({
                  colIndex: detail!.col,
                  rowIndex: detail!.row,
                  setIndex: detail!.set,
                  format,
                })
            "
            @click.stop
          />

          <CopyButton
            v-if="!disallowCopyingData"
            quaternary
            size="small"
            :text="false"
            :content="copyContent"
          />
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
import { onKeyStroke, useLocalStorage } from "@vueuse/core";
import {
  BracesIcon,
  ChevronDownIcon,
  ChevronUpIcon,
  WrapTextIcon,
} from "lucide-vue-next";
import { NButton, NPopover, NScrollbar, NTooltip } from "naive-ui";
import { computed } from "vue";
import { CopyButton, DrawerContent } from "@/components/v2";
import type { QueryResult } from "@/types/proto-es/v1/sql_service_pb";
import { useSQLResultViewContext } from "../context";
import BinaryFormatButton from "../DataTable/common/BinaryFormatButton.vue";
import {
  type BinaryFormat,
  useBinaryFormatContext,
} from "../DataTable/common/binary-format-store";
import { getPlainValue } from "../DataTable/common/utils";
import PrettyJSON from "./PrettyJSON.vue";

const props = defineProps<{
  result: QueryResult;
}>();

const { dark, detail, disallowCopyingData } = useSQLResultViewContext();
const { getBinaryFormat, setBinaryFormat } = useBinaryFormatContext();

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
  if (!detail.value) {
    return;
  }

  const { row, col } = detail.value;
  return props.result.rows[row].values[col];
});

const columnType = computed(() => {
  if (!detail.value) {
    return "";
  }
  return props.result.columnTypeNames[detail.value.col];
});

const binaryFormat = computed(() => {
  if (!detail.value) {
    return;
  }
  return getBinaryFormat({
    rowIndex: detail.value.row,
    colIndex: detail.value.col,
    setIndex: detail.value.set,
  });
});

// Check if the current value is binary data (using bytesValue)
const isBinaryData = computed(() => {
  if (!rawValue.value) return false;
  return rawValue.value.kind?.case === "bytesValue";
});

const content = computed(() => {
  return getPlainValue(rawValue.value, columnType.value, binaryFormat.value);
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
  return props.result.rows.length;
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

const copyContent = computed(() => {
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
});

const move = (offset: number) => {
  if (!detail.value) {
    return;
  }
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
</script>
