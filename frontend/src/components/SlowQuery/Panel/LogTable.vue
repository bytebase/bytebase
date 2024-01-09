<template>
  <NDataTable
    v-model:expanded-row-keys="expandedRowKeys"
    class="min-w-[120rem]"
    :columns="dataTableColumns"
    :data="slowQueryLogList"
    :max-height="'100%'"
    :row-key="rowKey"
    :row-props="rowProps"
    :virtual-scroll="true"
    :striped="true"
    :bordered="true"
    :bottom-bordered="true"
  >
    <template #empty>
      <div class="py-8 px-8 text-center">
        <p v-if="allowAdmin">
          {{ $t("slow-query.no-log-placeholder.admin") }}
        </p>
        <p v-else>
          {{ $t("slow-query.no-log-placeholder.developer") }}
        </p>
      </div>
    </template>
  </NDataTable>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { NDataTable, DataTableColumn } from "naive-ui";
import { computed, h, ref } from "vue";
import { useI18n } from "vue-i18n";
import HighlightCodeBlock from "@/components/HighlightCodeBlock";
import {
  ProjectV1Name,
  DatabaseV1Name,
  InstanceV1Name,
  EnvironmentV1Name,
} from "@/components/v2";
import type { ComposedSlowQueryLog } from "@/types";
import type { Duration } from "@/types/proto/google/protobuf/duration";
import { instanceV1HasSlowQueryDetail } from "@/utils";

const props = withDefaults(
  defineProps<{
    slowQueryLogList?: ComposedSlowQueryLog[];
    showPlaceholder?: boolean;
    showProjectColumn?: boolean;
    showEnvironmentColumn?: boolean;
    showInstanceColumn?: boolean;
    showDatabaseColumn?: boolean;
    allowAdmin: boolean;
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

const emit = defineEmits<{
  (event: "select", log: ComposedSlowQueryLog): void;
}>();

const { t } = useI18n();

const dataTableColumns = computed(
  (): DataTableColumn<ComposedSlowQueryLog>[] => {
    const columns: DataTableColumn<ComposedSlowQueryLog>[] = [
      {
        type: "expand",
        expandable: (_) => true,
        renderExpand: (item) =>
          h(
            "div",
            { class: "w-full max-h-[20rem] overflow-auto text-xs pl-2" },
            h(HighlightCodeBlock, {
              class: "whitespace-pre-wrap",
              code: item.log.statistics?.sqlFingerprint ?? "",
            })
          ),
      },
      {
        title: t("slow-query.sql-statement"),
        key: "statement",
        render: (item) =>
          h("div", { class: "truncate" }, item.log.statistics?.sqlFingerprint),
      },
      {
        title: t("slow-query.total-query-count"),
        key: "total_query_count",
        sorter: (rowA, rowB) =>
          (rowA.log.statistics?.count ?? 0) - (rowB.log.statistics?.count ?? 0),
        render: (item) => item.log.statistics?.count,
      },
      {
        title: t("slow-query.query-count-percent"),
        key: "query_count_percentage",
        render: (item) =>
          `${((item.log.statistics?.countPercent ?? 0) * 100).toFixed(2)}%`,
      },
      {
        title: t("slow-query.max-query-time"),
        key: "max_query_time",
        sorter: (rowA, rowB) =>
          durationSeconds(rowA.log.statistics?.maximumQueryTime) -
          durationSeconds(rowB.log.statistics?.maximumQueryTime),
        render: (item) => durationText(item.log.statistics?.maximumQueryTime),
      },
      {
        title: t("slow-query.avg-query-time"),
        key: "avg_query_time",
        sorter: (rowA, rowB) =>
          durationSeconds(rowA.log.statistics?.averageQueryTime) -
          durationSeconds(rowB.log.statistics?.averageQueryTime),
        render: (item) => durationText(item.log.statistics?.averageQueryTime),
      },
      {
        title: t("slow-query.query-time-percent"),
        key: "query_time_percentage",
        render: (item) =>
          `${((item.log.statistics?.queryTimePercent ?? 0) * 100).toFixed(2)}%`,
      },
      {
        title: t("slow-query.max-rows-examined"),
        key: "max_rows_examined",
        sorter: (rowA, rowB) =>
          (rowA.log.statistics?.maximumRowsExamined ?? 0) -
          (rowB.log.statistics?.maximumRowsExamined ?? 0),
        render: (item) =>
          instanceV1HasSlowQueryDetail(item.database.instanceEntity)
            ? item.log.statistics?.maximumRowsExamined
            : "-",
      },
      {
        title: t("slow-query.avg-rows-examined"),
        key: "avg_rows_examined",
        sorter: (rowA, rowB) =>
          (rowA.log.statistics?.averageRowsExamined ?? 0) -
          (rowB.log.statistics?.averageRowsExamined ?? 0),
        render: (item) =>
          instanceV1HasSlowQueryDetail(item.database.instanceEntity)
            ? item.log.statistics?.averageRowsExamined
            : "-",
      },
      {
        title: t("slow-query.max-rows-sent"),
        key: "max_rows_sent",
        sorter: (rowA, rowB) =>
          (rowA.log.statistics?.maximumRowsSent ?? 0) -
          (rowB.log.statistics?.maximumRowsSent ?? 0),
        render: (item) =>
          instanceV1HasSlowQueryDetail(item.database.instanceEntity)
            ? item.log.statistics?.maximumRowsSent
            : "-",
      },
      {
        title: t("slow-query.avg-rows-sent"),
        key: "avg_rows_sent",
        sorter: (rowA, rowB) =>
          (rowA.log.statistics?.averageRowsSent ?? 0) -
          (rowB.log.statistics?.averageRowsSent ?? 0),
        render: (item) => item.log.statistics?.averageRowsSent,
      },
    ];

    if (props.showProjectColumn) {
      columns.push({
        title: t("common.project"),
        key: "project",
        render: (item) =>
          h(ProjectV1Name, {
            project: item.database.projectEntity,
            link: false,
          }),
      });
    }
    if (props.showEnvironmentColumn) {
      columns.push({
        title: t("common.environment"),
        key: "environment",
        render: (item) =>
          h(EnvironmentV1Name, {
            environment: item.database.effectiveEnvironmentEntity,
            link: false,
          }),
      });
    }
    if (props.showInstanceColumn) {
      columns.push({
        title: t("common.instance"),
        key: "instance",
        render: (item) =>
          h(InstanceV1Name, {
            instance: item.database.instanceEntity,
            link: false,
          }),
      });
    }
    if (props.showDatabaseColumn) {
      columns.push({
        title: t("common.database"),
        key: "database",
        render: (item) =>
          h(DatabaseV1Name, {
            database: item.database,
            link: false,
          }),
      });
    }

    columns.push({
      title: t("slow-query.last-query-time"),
      key: "last_query_time",
      sorter: (rowA, rowB) =>
        (rowA.log.statistics?.latestLogTime?.getTime() ?? 0) -
        (rowB.log.statistics?.latestLogTime?.getTime() ?? 0),
      render: (item) =>
        dayjs(item.log.statistics?.latestLogTime).format("YYYY-MM-DD HH:mm:ss"),
    });

    return columns;
  }
);

const rowKey = (item: ComposedSlowQueryLog) => {
  return item.id;
};

const expandedRowKeys = ref<string[]>([]);

const durationSeconds = (duration: Duration | undefined) => {
  if (!duration) return 0;
  const { seconds, nanos } = duration;
  return seconds.toNumber() + nanos / 1e9;
};

const durationText = (duration: Duration | undefined) => {
  if (!duration) return "-";
  const total = durationSeconds(duration);
  return total.toFixed(2) + "s";
};

const rowProps = (row: ComposedSlowQueryLog) => {
  return {
    style: "cursor: pointer;",
    onClick: () => {
      emit("select", row);
    },
  };
};
</script>
