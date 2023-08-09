<template>
  <BBGrid
    :column-list="columns"
    :data-source="slowQueryLogList"
    :show-placeholder="showPlaceholder"
    :is-row-clickable="(row) => true"
    :is-row-expanded="isSelectedRow"
    class="border w-auto overflow-x-auto"
    header-class="capitalize"
    @click-row="(log: ComposedSlowQueryLog) => $emit('select', log)"
  >
    <template #item="{ item }: SlowQueryLogRow">
      <template v-if="item.log.statistics">
        <div class="bb-grid-cell whitespace-nowrap !pl-1 !pr-1">
          <NButton quaternary size="tiny" @click.stop="toggleExpandRow(item)">
            <heroicons:chevron-right
              class="w-4 h-4 transition-transform duration-150 cursor-pointer"
              :class="[isSelectedRow(item) && 'rotate-90']"
            />
          </NButton>
        </div>
        <div class="bb-grid-cell text-xs font-mono">
          <div class="truncate">
            {{ item.log.statistics.sqlFingerprint }}
          </div>
        </div>
        <div class="bb-grid-cell">
          {{ item.log.statistics.count }}
        </div>
        <div class="bb-grid-cell">
          {{ (item.log.statistics.countPercent * 100).toFixed(2) }}%
        </div>
        <div class="bb-grid-cell">
          {{ durationText(item.log.statistics.maximumQueryTime) }}
        </div>
        <div class="bb-grid-cell">
          {{ durationText(item.log.statistics.averageQueryTime) }}
        </div>
        <div class="bb-grid-cell">
          {{ (item.log.statistics.queryTimePercent * 100).toFixed(2) }}%
        </div>
        <div class="bb-grid-cell">
          {{
            instanceV1HasSlowQueryDetail(item.database.instanceEntity)
              ? item.log.statistics.maximumRowsExamined
              : "-"
          }}
        </div>
        <div class="bb-grid-cell">
          {{
            instanceV1HasSlowQueryDetail(item.database.instanceEntity)
              ? item.log.statistics.averageRowsExamined
              : "-"
          }}
        </div>
        <div class="bb-grid-cell">
          {{
            instanceV1HasSlowQueryDetail(item.database.instanceEntity)
              ? item.log.statistics.maximumRowsSent
              : "-"
          }}
        </div>
        <div class="bb-grid-cell">
          {{ item.log.statistics.averageRowsSent }}
        </div>
        <div v-if="showProjectColumn" class="bb-grid-cell">
          <ProjectV1Name :project="item.database.projectEntity" :link="false" />
        </div>
        <div v-if="showEnvironmentColumn" class="bb-grid-cell">
          <EnvironmentV1Name
            :environment="item.database.instanceEntity.environmentEntity"
            :link="false"
          />
        </div>
        <div v-if="showInstanceColumn" class="bb-grid-cell">
          <InstanceV1Name
            :instance="item.database.instanceEntity"
            :link="false"
          />
        </div>
        <div v-if="showDatabaseColumn" class="bb-grid-cell">
          <DatabaseV1Name :database="item.database" :link="false" />
        </div>
        <div class="bb-grid-cell whitespace-nowrap !pr-4">
          {{
            dayjs(item.log.statistics.latestLogTime).format(
              "YYYY-MM-DD HH:mm:ss"
            )
          }}
        </div>
      </template>
    </template>

    <template #expanded-item="{ item }: SlowQueryLogRow">
      <div class="w-full max-h-[20rem] overflow-auto text-xs pl-2">
        <HighlightCodeBlock
          :code="item.log.statistics?.sqlFingerprint"
          class="whitespace-pre-wrap"
        />
      </div>
    </template>

    <template #placeholder-content>
      <div class="py-8 px-8 text-center">
        <p v-if="allowAdmin">
          {{ $t("slow-query.no-log-placeholder.admin") }}
        </p>
        <p v-else>
          {{ $t("slow-query.no-log-placeholder.developer") }}
        </p>
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { computed, shallowRef, watch } from "vue";
import { useI18n } from "vue-i18n";
import { type BBGridColumn, type BBGridRow, BBGrid } from "@/bbkit";
import {
  DatabaseV1Name,
  InstanceV1Name,
  EnvironmentV1Name,
} from "@/components/v2";
import type { ComposedSlowQueryLog } from "@/types";
import type { Duration } from "@/types/proto/google/protobuf/duration";
import { instanceV1HasSlowQueryDetail } from "@/utils";

export type SlowQueryLogRow = BBGridRow<ComposedSlowQueryLog>;

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

defineEmits<{
  (event: "select", log: ComposedSlowQueryLog): void;
}>();

const { t } = useI18n();
const selectedSlowQueryLog = shallowRef<ComposedSlowQueryLog>();

const columns = computed(() => {
  const columns = [
    {
      width: "auto",
    },
    {
      title: t("slow-query.sql-statement"),
      width: "minmax(10rem, 1fr)",
    },
    {
      title: t("slow-query.total-query-count"),
      width: "6rem",
    },
    {
      title: t("slow-query.query-count-percent"),
      width: "6rem",
    },
    {
      title: t("slow-query.max-query-time"),
      width: "6rem",
    },
    {
      title: t("slow-query.avg-query-time"),
      width: "6rem",
    },
    {
      title: t("slow-query.query-time-percent"),
      width: "6rem",
    },
    {
      title: t("slow-query.max-rows-examined"),
      width: "6rem",
    },
    {
      title: t("slow-query.avg-rows-examined"),
      width: "6rem",
    },
    {
      title: t("slow-query.max-rows-sent"),
      width: "6rem",
    },
    {
      title: t("slow-query.avg-rows-sent"),
      width: "6rem",
    },
    props.showProjectColumn && {
      title: t("common.project"),
      width: "minmax(6rem, auto)",
    },
    props.showEnvironmentColumn && {
      title: t("common.environment"),
      width: "8rem",
    },
    props.showInstanceColumn && {
      title: t("common.instance"),
      width: "12rem",
    },
    props.showDatabaseColumn && {
      title: t("common.database"),
      width: "8rem",
    },
    {
      title: t("slow-query.last-query-time"),
      width: "auto",
    },
  ].filter((col) => !!col) as BBGridColumn[];
  return columns;
});

const isSelectedRow = (item: ComposedSlowQueryLog) => {
  return selectedSlowQueryLog.value === item;
};

const toggleExpandRow = (item: ComposedSlowQueryLog) => {
  if (selectedSlowQueryLog.value === item) {
    selectedSlowQueryLog.value = undefined;
  } else {
    selectedSlowQueryLog.value = item;
  }
};

const durationText = (duration: Duration | undefined) => {
  if (!duration) return "-";
  const { seconds, nanos } = duration;
  const total = seconds + nanos / 1e9;
  return total.toFixed(2) + "s";
};

watch(
  () => props.slowQueryLogList,
  () => {
    selectedSlowQueryLog.value = undefined;
  },
  { immediate: true }
);
</script>
