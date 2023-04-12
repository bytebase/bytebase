<template>
  <BBGrid
    :column-list="columns"
    :data-source="slowQueryLogList"
    :show-placeholder="showPlaceholder"
    class="border compact w-auto overflow-x-auto"
    header-class="capitalize"
    @click-row="(log: ComposedSlowQueryLog) => $emit('select', log)"
  >
    <template #item="{ item: { log, database } }: SlowQueryLogRow">
      <template v-if="log.statistics">
        <div class="bb-grid-cell">
          <HumanizeTs :ts="dateToTS(log.statistics.latestLogTime)" />
        </div>
        <div class="bb-grid-cell">
          <InstanceName :instance="database.instance" />
        </div>
        <div class="bb-grid-cell">
          <DatabaseName :database="database" />
        </div>
        <div class="bb-grid-cell text-xs font-mono">
          <div class="truncate">
            {{ log.statistics.sqlFingerprint }}
          </div>
        </div>
        <div class="bb-grid-cell">
          {{ log.statistics.count }}
        </div>
        <div class="bb-grid-cell">
          {{
            log.statistics.nightyFifthPercentileQueryTime?.seconds.toFixed(6)
          }}
        </div>
        <div class="bb-grid-cell">
          {{ log.statistics.averageQueryTime?.seconds.toFixed(6) }}
        </div>
        <div class="bb-grid-cell">
          {{ log.statistics.nightyFifthPercentileRowsExamined }}
        </div>
        <div class="bb-grid-cell">
          {{ log.statistics.averageRowsExamined }}
        </div>
        <div class="bb-grid-cell">
          {{ log.statistics.nightyFifthPercentileRowsSent }}
        </div>
        <div class="bb-grid-cell">
          {{ log.statistics.averageRowsSent }}
        </div>
      </template>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import { type BBGridColumn, type BBGridRow, BBGrid } from "@/bbkit";
import type { ComposedSlowQueryLog } from "@/types";
import HumanizeTs from "@/components/misc/HumanizeTs.vue";
import { DatabaseName, InstanceName } from "@/components/v2";

export type SlowQueryLogRow = BBGridRow<ComposedSlowQueryLog>;

withDefaults(
  defineProps<{
    slowQueryLogList?: ComposedSlowQueryLog[];
    showPlaceholder?: boolean;
  }>(),
  {
    slowQueryLogList: () => [],
    showPlaceholder: true,
  }
);

defineEmits<{
  (event: "select", log: ComposedSlowQueryLog): void;
}>();

const { t } = useI18n();

const columns = computed(() => {
  const columns: BBGridColumn[] = [
    {
      title: t("slow-query.last-query-time"),
      width: "minmax(8rem, auto)",
    },
    {
      title: t("common.instance"),
      width: "minmax(12rem, 18rem)",
    },
    {
      title: t("common.database"),
      width: "minmax(12rem, 18rem)",
    },
    {
      title: t("slow-query.sql-statement"),
      width: "minmax(20rem, 1fr)",
    },
    {
      title: t("slow-query.total-query-count"),
      width: "minmax(6rem, auto)",
    },
    {
      title: t("slow-query.query-time-95-percent"),
      width: "minmax(6rem, auto)",
    },
    {
      title: t("slow-query.query-time-avg"),
      width: "minmax(6rem, auto)",
    },
    {
      title: t("slow-query.rows-examined-95-percent"),
      width: "minmax(6rem, auto)",
    },
    {
      title: t("slow-query.rows-sent-95-percent"),
      width: "minmax(6rem, auto)",
    },
    {
      title: t("slow-query.rows-examined-avg"),
      width: "minmax(6rem, auto)",
    },
    {
      title: t("slow-query.rows-sent-avg"),
      width: "minmax(6rem, auto)",
    },
  ];
  return columns;
});

const dateToTS = (date: Date | undefined) => {
  if (!date) return 0;
  return Math.floor(date.getTime() / 1000);
};
</script>
