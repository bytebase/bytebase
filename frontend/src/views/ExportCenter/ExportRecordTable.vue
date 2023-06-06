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
      <div class="bb-grid-cell">#{{ item.issueId }}</div>
      <div class="bb-grid-cell">
        {{ item.database.databaseName }}
      </div>
      <div class="bb-grid-cell">
        {{ item.database.instanceEntity.environmentEntity.title }}
      </div>
      <div class="bb-grid-cell">
        {{ item.database.instanceEntity.title }}
      </div>
      <div class="bb-grid-cell">
        {{ item.database.projectEntity.title }}
      </div>
      <div class="bb-grid-cell">
        {{ item.expiration ? humanizeDate(new Date(item.expiration)) : "*" }}
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
import { computed, shallowRef } from "vue";
import { useI18n } from "vue-i18n";
import { ExportRecord } from "./types";
import { BBGridColumn } from "@/bbkit";
import ExportDataButton from "./ExportDataButton.vue";

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
    { title: "Issue", width: "1fr" },
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
      title: t("common.expiration"),
      width: "1fr",
    },
    { title: "", width: "1fr" },
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
</script>
