<template>
  <NDrawer
    :show="slowQueryLog !== undefined"
    :auto-focus="false"
    width="auto"
    @update:show="(show) => !show && $emit('close')"
  >
    <NDrawerContent
      :title="$t('common.detail')"
      :closable="true"
      class="w-[calc(100vw-8rem)] lg:max-w-[56rem]"
    >
      <div v-if="slowQueryLog" class="max-h-full flex flex-col gap-y-4 text-sm">
        <div
          class="grid grid-cols-[auto_1fr] md:grid-cols-[minmax(auto,7rem)_1fr_minmax(auto,7rem)_1fr_minmax(auto,7rem)_1fr] gap-x-2 gap-y-4"
        >
          <div class="contents">
            <label class="font-medium">{{ $t("common.environment") }}</label>

            <EnvironmentName :environment="database.instance.environment" />
          </div>

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
              class="col-start-2 md:col-span-5 max-h-[8rem] overflow-auto py-0.5 text-xs"
            >
              <HighlightCodeBlock
                :code="log.statistics?.sqlFingerprint ?? ''"
                class="whitespace-pre-wrap"
              />
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
            <template #item="{ item: detail, row }: SlowQueryDetailsRow">
              <div class="bb-grid-cell whitespace-nowrap !pl-4 !pr-2">
                {{ row + 1 }}
              </div>
              <div class="bb-grid-cell whitespace-nowrap !pr-4">
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
    </NDrawerContent>
  </NDrawer>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { NDrawer, NDrawerContent } from "naive-ui";

import { type BBGridColumn, type BBGridRow, BBGrid } from "@/bbkit";
import type { ComposedSlowQueryLog } from "@/types";
import type { SlowQueryDetails } from "@/types/proto/v1/database_service";
import { DatabaseName, InstanceName, EnvironmentName } from "@/components/v2";
import HighlightCodeBlock from "@/components/HighlightCodeBlock";

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
</script>
