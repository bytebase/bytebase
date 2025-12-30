<template>
  <PagedTable
    ref="auditPagedTable"
    :session-key="`bb.page-audit-log-table.settings-audit-log-v1-table.${parent}`"
    :fetch-list="fetchAuditLog"
    :order-keys="['create_time']"
  >
    <template #table="{ list, loading, sorters, onSortersUpdate }">
      <AuditLogDataTable
        :key="`audit-log-table.${parent}`"
        :audit-log-list="list"
        :show-project="false"
        :loading="loading"
        :sorters="sorters"
        @update:sorters="onSortersUpdate"
      />
    </template>
  </PagedTable>
</template>

<script lang="tsx" setup>
import dayjs from "dayjs";
import { ref, watch } from "vue";
// https://github.com/vuejs/language-tools/issues/3206
import type { ComponentExposed } from "vue-component-type-helpers";
import AuditLogDataTable from "@/components/AuditLog/AuditLogDataTable.vue";
import type {
  DownloadContent,
  ExportOption,
} from "@/components/DataExportButton.vue";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { useAuditLogStore } from "@/store";
import type { AuditLogFilter } from "@/types";
import type { AuditLog } from "@/types/proto-es/v1/audit_log_service_pb";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";

const props = defineProps<{
  parent: string;
  filter: AuditLogFilter;
}>();

const auditLogStore = useAuditLogStore();
const auditPagedTable = ref<ComponentExposed<typeof PagedTable<AuditLog>>>();

const fetchAuditLog = async ({
  pageToken,
  pageSize,
  orderBy,
}: {
  pageToken: string;
  pageSize: number;
  orderBy?: string;
}) => {
  const { nextPageToken, auditLogs } = await auditLogStore.fetchAuditLogs({
    pageToken,
    pageSize,
    parent: props.parent,
    filter: props.filter,
    orderBy,
  });
  return { nextPageToken, list: auditLogs };
};

watch(
  [() => props.filter, () => props.parent],
  () => auditPagedTable.value?.refresh(),
  { deep: true }
);

const handleExport = async ({
  options,
  resolve,
  reject,
}: {
  options: ExportOption;
  reject: (reason?: unknown) => void;
  resolve: (content: DownloadContent[]) => void;
}) => {
  let pageToken = "";
  let i = 0;
  const contents: DownloadContent[] = [];

  try {
    while (i === 0 || pageToken !== "") {
      i++;
      const { content, nextPageToken } = await auditLogStore.exportAuditLogs({
        search: {
          parent: props.parent,
          filter: props.filter,
          orderBy: auditPagedTable.value?.orderBy,
          pageSize: 5000, // The maximum page size is 5000
          pageToken,
        },
        format: options.format,
      });
      pageToken = nextPageToken;
      contents.push({
        content,
        filename: `audit-log.file${i}.${dayjs(new Date()).format("YYYY-MM-DDTHH-mm-ss")}.${ExportFormat[options.format].toLowerCase()}`,
      });
    }
    resolve(contents);
  } catch (err) {
    reject(err);
  }
};

defineExpose({
  handleExport,
});
</script>