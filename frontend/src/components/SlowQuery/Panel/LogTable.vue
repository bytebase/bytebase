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
        <div v-if="showProjectColumn" class="bb-grid-cell">
          <ProjectName :project="database.project" :link="false" />
        </div>
        <div v-if="showEnvironmentColumn" class="bb-grid-cell">
          <EnvironmentName
            :environment="database.instance.environment"
            :link="false"
          />
        </div>
        <div v-if="showInstanceColumn" class="bb-grid-cell">
          <InstanceName :instance="database.instance" :link="false" />
        </div>
        <div v-if="showDatabaseColumn" class="bb-grid-cell">
          <DatabaseName :database="database" :link="false" />
        </div>
        <div class="bb-grid-cell whitespace-nowrap !pr-4">
          {{
            dayjs(log.statistics.latestLogTime).format("YYYY-MM-DD HH:mm:ss")
          }}
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
import { DatabaseName, InstanceName, EnvironmentName } from "@/components/v2";

export type SlowQueryLogRow = BBGridRow<ComposedSlowQueryLog>;

const props = withDefaults(
  defineProps<{
    slowQueryLogList?: ComposedSlowQueryLog[];
    showPlaceholder?: boolean;
    showProjectColumn?: boolean;
    showEnvironmentColumn?: boolean;
    showInstanceColumn?: boolean;
    showDatabaseColumn?: boolean;
  }>(),
  {
    slowQueryLogList: () => [],
    showPlaceholder: true,
    showProjectColumn: true,
    showEnvironmentColumn: true,
    showInstanceColumn: true,
    showDatabaseColumn: true,
  }
);

defineEmits<{
  (event: "select", log: ComposedSlowQueryLog): void;
}>();

const { t } = useI18n();

const columns = computed(() => {
  const columns = [
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
      title: t("slow-query.rows-examined-avg"),
      width: "minmax(6rem, auto)",
    },
    {
      title: t("slow-query.rows-sent-95-percent"),
      width: "minmax(6rem, auto)",
    },
    {
      title: t("slow-query.rows-sent-avg"),
      width: "minmax(6rem, auto)",
    },
    props.showProjectColumn && {
      title: t("common.project"),
      width: "minmax(6rem, auto)",
    },
    props.showEnvironmentColumn && {
      title: t("common.environment"),
      width: "minmax(6rem, auto)",
    },
    props.showInstanceColumn && {
      title: t("common.instance"),
      width: "minmax(12rem, 18rem)",
    },
    props.showDatabaseColumn && {
      title: t("common.database"),
      width: "minmax(12rem, 18rem)",
    },
    {
      title: t("slow-query.last-query-time"),
      width: "auto",
    },
  ].filter((col) => !!col) as BBGridColumn[];
  return columns;
});
</script>
