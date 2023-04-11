<template>
  <BBModal :title="$t('common.detail')" @close="$emit('close')">
    <div
      class="max-h-[calc(100vh-12rem)] w-[calc(100vw-8rem)] lg:max-w-[56rem] flex flex-col gap-y-4 text-sm"
    >
      <div
        class="grid grid-cols-[minmax(auto,7rem)_1fr] md:grid-cols-[minmax(auto,7rem)_1fr_minmax(auto,7rem)_1fr] gap-x-2 gap-y-4"
      >
        <div class="contents">
          <label class="font-medium">{{ $t("common.instance") }}</label>

          <InstanceName :instance="database.instance" />
        </div>

        <div class="contents">
          <label class="font-medium">{{ $t("common.database") }}</label>

          <DatabaseName :database="database" />
        </div>

        <div class="contents">
          <label class="font-medium whitespace-nowrap">
            {{ $t("common.sql-statement") }}
          </label>

          <div
            class="col-start-2 md:col-span-3 max-h-[8rem] overflow-auto border p-1 text-xs"
          >
            <HighlightCodeBlock :code="log.statistics?.sqlFingerprint ?? ''" />
          </div>
        </div>
      </div>
      <div class="flex-1 overflow-auto border">
        <BBGrid
          :column-list="columns"
          :data-source="log.statistics?.samples"
          :row-clickable="false"
          class="compact"
          header-class="capitalize"
        >
          <template #item="{ item: detail }: SlowQueryDetailsRow">
            <div class="bb-grid-cell">
              {{ dayjs(detail.startTime).format("YYYY-MM-DD HH:mm:ss") }}
            </div>
            <div class="bb-grid-cell">
              {{ detail.queryTime?.seconds.toFixed(6) }}
            </div>
            <div class="bb-grid-cell">
              {{ detail.lockTime?.seconds.toFixed(6) }}
            </div>
            <div class="bb-grid-cell">
              {{ detail.rowsExamined }}
            </div>
            <div class="bb-grid-cell">
              {{ detail.rowsSent }}
            </div>
          </template>
        </BBGrid>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import { type BBGridColumn, type BBGridRow, BBModal, BBGrid } from "@/bbkit";
import type { ComposedSlowQueryLog } from "@/types";
import type { SlowQueryDetails } from "@/types/proto/v1/database_service";
import { DatabaseName, InstanceName } from "@/components/v2";
import HighlightCodeBlock from "@/components/HighlightCodeBlock";

export type SlowQueryDetailsRow = BBGridRow<SlowQueryDetails>;

const props = defineProps<{
  slowQueryLog: ComposedSlowQueryLog;
}>();

defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();

const columns = computed(() => {
  const columns: BBGridColumn[] = [
    {
      title: t("slow-query.query-start-time"),
      width: "minmax(auto, 2fr)",
    },
    {
      title: t("slow-query.query-time"),
      width: "minmax(auto, 1fr)",
    },
    {
      title: t("slow-query.lock-time"),
      width: "minmax(auto, 1fr)",
    },
    {
      title: t("slow-query.rows-examined"),
      width: "minmax(auto, 1fr)",
    },
    {
      title: t("slow-query.rows-sent"),
      width: "minmax(auto, 1fr)",
    },
  ];
  return columns;
});
const log = computed(() => props.slowQueryLog.log);
const database = computed(() => props.slowQueryLog.database);
</script>
