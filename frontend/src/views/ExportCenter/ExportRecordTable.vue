<template>
  <BBGrid
    :column-list="COLUMN_LIST"
    :data-source="exportRecords"
    :row-clickable="true"
    :show-placeholder="true"
    :is-row-expanded="isSelectedRow"
    class="border w-auto overflow-x-auto"
    header-class="capitalize"
  >
    <template #item="{ item }: { item: ExportRecord }">
      <div class="bb-grid-cell">
        <NButton quaternary size="tiny" @click.stop="toggleExpandRow(item)">
          <heroicons:chevron-right
            class="w-4 h-auto transition-transform duration-150 cursor-pointer"
            :class="[isSelectedRow(item) && 'rotate-90']"
          />
        </NButton>
      </div>
      <div
        class="bb-grid-cell text-blue-600 hover:underline"
        @click="gotoIssuePage(item)"
      >
        {{ `#${item.issueId}` }}
      </div>
      <div class="bb-grid-cell">
        {{ item.database.databaseName }}
      </div>
      <div class="bb-grid-cell">
        <EnvironmentV1Name
          :environment="item.database.instanceEntity.environmentEntity"
        />
      </div>
      <div class="bb-grid-cell">
        <InstanceV1Name :instance="item.database.instanceEntity" />
      </div>
      <div class="bb-grid-cell">
        <ProjectV1Name :project="item.database.projectEntity" />
      </div>
      <div class="bb-grid-cell">
        {{ item.maxRowCount }}
      </div>
      <div class="bb-grid-cell">
        {{
          item.expiration ? dayjs(new Date(item.expiration)).format("LLL") : "*"
        }}
      </div>
      <div class="bb-grid-cell">
        <ExportDataButton :export-record="item" />
      </div>
    </template>
    <template #expanded-item="{ item }: { item: ExportRecord }">
      <div class="w-full max-h-[20rem] overflow-auto text-xs pl-2">
        <HighlightCodeBlock
          :code="item.statement"
          class="whitespace-pre-wrap"
        />
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { computed, shallowRef } from "vue";
import { useI18n } from "vue-i18n";
import { BBGridColumn } from "@/bbkit";
import { InstanceV1Name } from "@/components/v2";
import { pushNotification, useIssueStore } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { issueSlug } from "@/utils";
import ExportDataButton from "./ExportDataButton.vue";
import { ExportRecord } from "./types";

defineProps<{
  exportRecords: ExportRecord[];
}>();

defineEmits<{
  (event: "export", exportRecord: ExportRecord): void;
}>();

const { t } = useI18n();

const selectedExportRecord = shallowRef<ExportRecord>();

const COLUMN_LIST = computed(() => {
  const columns: BBGridColumn[] = [
    { title: "", width: "4rem" },
    { title: t("common.issue"), width: "5rem" },
    {
      title: t("common.database"),
      width: "1fr",
    },
    {
      title: t("common.environment"),
      width: "1fr",
    },
    {
      title: t("common.instance"),
      width: "1fr",
    },
    {
      title: t("common.project"),
      width: "1fr",
    },
    {
      title: t("issue.grant-request.export-rows"),
      width: "1fr",
    },
    {
      title: t("common.expiration"),
      width: "1fr",
    },
    { title: "", width: "6rem" },
  ];

  return columns;
});

const isSelectedRow = (item: ExportRecord) => {
  return selectedExportRecord.value === item;
};

const toggleExpandRow = (item: ExportRecord) => {
  if (selectedExportRecord.value === item) {
    selectedExportRecord.value = undefined;
  } else {
    selectedExportRecord.value = item;
  }
};

const gotoIssuePage = async (item: ExportRecord) => {
  const issue = await useIssueStore().getOrFetchIssueById(item.issueId);
  if (issue.id === UNKNOWN_ID) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Issue #${issue.id} not found`,
    });
    return;
  }

  window.open(`/issue/${issueSlug(issue.name, issue.id)}`, "_blank");
};
</script>
