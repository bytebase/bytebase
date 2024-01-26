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

        <div>
          <NButton v-if="!disallowCopyingData" size="small" @click="handleCopy">
            <template #icon>
              <ClipboardIcon class="w-4 h-4" />
            </template>
            {{ $t("common.copy") }}
          </NButton>
        </div>
      </div>
      <!-- eslint-disable vue/no-v-html -->
      <div
        class="flex-1 overflow-auto whitespace-pre-wrap text-sm font-mono border p-2"
        :class="disallowCopyingData && 'select-none'"
        v-html="html"
      ></div>
    </div>
  </DrawerContent>
</template>

<script setup lang="ts">
import { onKeyStroke, useClipboard } from "@vueuse/core";
import { escape } from "lodash-es";
import { ChevronDownIcon, ChevronUpIcon, ClipboardIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { DrawerContent } from "@/components/v2";
import { pushNotification } from "@/store";
import { useSQLResultViewContext } from "./context";

const { t } = useI18n();
const { dark, detail, disallowCopyingData } = useSQLResultViewContext();

const detailValue = computed(() => {
  const { row, col, table } = detail.value;
  if (!table) return undefined;

  const value = table
    .getPrePaginationRowModel()
    .rows[row]?.getVisibleCells()
    [col]?.getValue<string>();
  return value;
});

const totalCount = computed(() => {
  const { table } = detail.value;
  if (!table) return 0;
  return table.getPrePaginationRowModel().rows.length;
});

const html = computed(() => {
  const str = detailValue.value;
  if (!str || str.length === 0) {
    return `<br style="min-width: 1rem; display: inline-flex;" />`;
  }

  return escape(str);
});

const { copy, copied } = useClipboard({
  source: computed(() => {
    return detailValue.value ?? "";
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
</script>
