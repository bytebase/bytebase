<template>
  <DrawerContent
    :title="$t('common.detail')"
    class="w-[100vw-4rem] min-w-[20rem] max-w-[100vw-4rem] md:w-[33vw]"
  >
    <div
      class="h-full flex flex-col gap-y-2"
      :class="dark ? 'text-white' : 'text-main'"
    >
      <div class="flex items-center justify-between gap-x-4">
        <div class="flex items-center gap-x-2">
          <NButton size="tiny" :disabled="detail.row === 0" @click="move(-1)">
            <template #icon>
              <heroicons:chevron-up />
            </template>
          </NButton>
          <NButton
            size="tiny"
            :disabled="detail.row === totalCount - 1"
            @click="move(1)"
          >
            <template #icon>
              <heroicons:chevron-down />
            </template>
          </NButton>
          <div class="text-xs text-control-light flex items-center gap-x-1">
            <span>{{ detail.row + 1 }}</span>
            <span>/</span>
            <span>{{ totalCount }}</span>
          </div>
        </div>

        <div>
          <NButton
            v-if="!disallowCopyingData"
            size="small"
            type="primary"
            @click="handleCopy"
          >
            {{ $t("common.copy") }}
          </NButton>
        </div>
      </div>
      <!-- eslint-disable vue/no-v-html -->
      <div
        class="flex-1 overflow-auto whitespace-pre-wrap text-sm"
        :class="disallowCopyingData && 'select-none'"
        v-html="html"
      ></div>
    </div>
  </DrawerContent>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { escape, get } from "lodash-es";
import { NButton } from "naive-ui";
import { onKeyStroke, useClipboard } from "@vueuse/core";
import { useI18n } from "vue-i18n";

import { DrawerContent } from "@/components/v2";
import { useSQLResultViewContext } from "./context";
import { SQLResultSetV1 } from "@/types";
import { extractSQLRowValue } from "@/utils";
import { QueryResult, RowValue } from "@/types/proto/v1/sql_service";
import { pushNotification } from "@/store";

const props = defineProps<{
  resultSet?: SQLResultSetV1;
}>();

const { t } = useI18n();
const { dark, detail, disallowCopyingData } = useSQLResultViewContext();

const value = computed(() => {
  const { resultSet } = props;
  const { set, row, col } = detail.value;
  const cell: RowValue =
    get(resultSet, `results.${set}.rows.${row}.values.${col}`) ??
    RowValue.fromJSON({});
  return extractSQLRowValue(cell);
});

const totalCount = computed(() => {
  const result: QueryResult =
    get(props.resultSet, `results.${detail.value.set}`) ??
    QueryResult.fromJSON({});
  return result.rows.length;
});

const html = computed(() => {
  const str = String(value.value);
  if (str.length === 0) {
    return `<br style="min-width: 1rem; display: inline-flex;" />`;
  }

  return escape(str);
});

const { copy, copied } = useClipboard({
  source: computed(() => {
    return String(value.value);
  }),
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
