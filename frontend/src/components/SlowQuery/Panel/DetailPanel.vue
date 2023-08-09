<template>
  <NDrawer
    :show="slowQueryLog !== undefined"
    :auto-focus="false"
    width="auto"
    :z-index="30"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <NDrawerContent
      :title="$t('slow-query.detail')"
      :closable="true"
      class="w-[calc(100vw-2rem)] lg:max-w-[64rem] xl:max-w-[72rem]"
    >
      <div v-if="slowQueryLog" class="max-h-full flex flex-col gap-y-4 text-sm">
        <div
          class="grid grid-cols-[auto_1fr] md:grid-cols-[minmax(auto,7rem)_1fr_minmax(auto,7rem)_1fr] gap-x-2 gap-y-4"
        >
          <div class="contents">
            <label class="font-medium capitalize">
              {{ $t("common.project") }}
            </label>

            <ProjectV1Name :project="database.projectEntity" />
          </div>

          <div class="contents">
            <label class="font-medium capitalize">
              {{ $t("common.environment") }}
            </label>

            <EnvironmentV1Name
              :environment="database.instanceEntity.environmentEntity"
            />
          </div>

          <div class="contents">
            <label class="font-medium capitalize">
              {{ $t("common.instance") }}
            </label>

            <InstanceV1Name :instance="database.instanceEntity" />
          </div>

          <div class="contents">
            <label class="font-medium capitalize">
              {{ $t("common.database") }}
            </label>

            <DatabaseV1Name :database="database" />
          </div>

          <div class="contents">
            <label class="font-medium capitalize whitespace-nowrap">
              {{ $t("common.sql-statement") }}
            </label>

            <div
              class="col-start-2 md:col-span-3 max-h-[8rem] overflow-auto py-0.5 text-xs"
            >
              <HighlightCodeBlock
                :code="log.statistics?.sqlFingerprint ?? ''"
                class="whitespace-pre-wrap"
              />
            </div>
          </div>
        </div>
        <IndexAdvisor v-if="slowQueryLog" :slow-query-log="slowQueryLog" />
        <div
          v-if="instanceV1HasSlowQueryDetail(database.instanceEntity)"
          class="flex-1 overflow-auto border"
        >
          <BBGrid
            :column-list="columns"
            :data-source="log.statistics?.samples"
            :row-clickable="false"
            :is-row-expanded="isSelectedRow"
            class="compact"
            header-class="capitalize"
          >
            <template #item="{ item: detail }: SlowQueryDetailsRow">
              <div class="bb-grid-cell whitespace-nowrap !pl-1 !pr-1">
                <NButton quaternary size="tiny" @click="selectRow(detail)">
                  <heroicons:chevron-right
                    class="w-4 h-4 transition-transform duration-150 cursor-pointer"
                    :class="[isSelectedRow(detail) && 'rotate-90']"
                  />
                </NButton>
              </div>
              <div class="bb-grid-cell whitespace-nowrap !pr-4">
                {{ dayjs(detail.startTime).format("YYYY-MM-DD HH:mm:ss") }}
              </div>
              <div class="bb-grid-cell text-xs font-mono">
                <div class="truncate">
                  {{ detail.sqlText }}
                </div>
              </div>
              <div class="bb-grid-cell">
                {{ detail.queryTime?.seconds.toFixed(2) }}s
              </div>
              <div class="bb-grid-cell">
                {{ detail.lockTime?.seconds.toFixed(2) }}s
              </div>
              <div class="bb-grid-cell">
                {{ detail.rowsExamined }}
              </div>
              <div class="bb-grid-cell">
                {{ detail.rowsSent }}
              </div>
            </template>

            <template #expanded-item="{ item: detail }: SlowQueryDetailsRow">
              <div class="w-full max-h-[20rem] overflow-auto text-xs pl-2">
                <HighlightCodeBlock
                  :code="detail.sqlText"
                  class="whitespace-pre-wrap"
                />
              </div>
            </template>
          </BBGrid>
        </div>
      </div>
    </NDrawerContent>
  </NDrawer>
</template>

<script lang="ts" setup>
import { NButton, NDrawer, NDrawerContent } from "naive-ui";
import { computed, shallowRef, watch } from "vue";
import { useI18n } from "vue-i18n";
import { type BBGridColumn, type BBGridRow, BBGrid } from "@/bbkit";
import HighlightCodeBlock from "@/components/HighlightCodeBlock";
import {
  DatabaseV1Name,
  InstanceV1Name,
  EnvironmentV1Name,
  ProjectV1Name,
} from "@/components/v2";
import type { ComposedSlowQueryLog } from "@/types";
import type { SlowQueryDetails } from "@/types/proto/v1/database_service";
import { instanceV1HasSlowQueryDetail } from "@/utils";
import IndexAdvisor from "./IndexAdvisor.vue";

export type SlowQueryDetailsRow = BBGridRow<SlowQueryDetails>;

const props = defineProps<{
  slowQueryLog: ComposedSlowQueryLog | undefined;
}>();

defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();

const columns = computed(() => {
  const columns: BBGridColumn[] = [
    {
      width: "auto",
    },
    {
      title: t("slow-query.query-start-time"),
      width: "auto",
    },
    {
      title: t("common.sql-statement"),
      width: "minmax(6rem, 3fr)",
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

const log = computed(() => props.slowQueryLog!.log);
const database = computed(() => props.slowQueryLog!.database);
const selectedDetail = shallowRef<SlowQueryDetails>();

const isSelectedRow = (item: SlowQueryDetails) => {
  return selectedDetail.value === item;
};

const selectRow = (item: SlowQueryDetails) => {
  if (selectedDetail.value === item) {
    selectedDetail.value = undefined;
  } else {
    selectedDetail.value = item;
  }
};

watch(
  () => props.slowQueryLog,
  () => {
    selectedDetail.value = undefined;
  },
  { immediate: true }
);
</script>
