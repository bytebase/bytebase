<template>
  <div class="w-full space-y-4 pb-6">
    <FeatureAttention feature="bb.feature.audit-log" />
    <AuditLogSearch v-model:params="state.params">
      <template #searchbox-suffix>
        <DataExportButton
          size="medium"
          :file-type="'raw'"
          :support-formats="[
            ExportFormat.CSV,
            ExportFormat.JSON,
            ExportFormat.XLSX,
          ]"
          :tooltip="disableExportTip"
          :disabled="!hasAuditLogFeature || !!disableExportTip"
          @export="handleExport"
        />
      </template>
    </AuditLogSearch>

    <PagedTable
      v-if="hasAuditLogFeature"
      ref="auditLogPagedTable"
      session-key="bb.page-audit-log-table.settings-audit-log-v1-table"
      :fetch-list="fetchAuditLog"
    >
      <template #table="{ list, loading }">
        <AuditLogDataTable
          key="audit-log-table"
          :loading="loading"
          :audit-log-list="list"
        />
      </template>
    </PagedTable>
    <NEmpty class="py-12 border rounded" v-else />
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { NEmpty } from "naive-ui";
import type { BinaryLike } from "node:crypto";
import { reactive, computed, watch, ref } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useI18n } from "vue-i18n";
import AuditLogDataTable from "@/components/AuditLog/AuditLogDataTable.vue";
import AuditLogSearch from "@/components/AuditLog/AuditLogSearch";
import { buildSearchAuditLogParams } from "@/components/AuditLog/AuditLogSearch/utils";
import type { ExportOption } from "@/components/DataExportButton.vue";
import DataExportButton from "@/components/DataExportButton.vue";
import { FeatureAttention } from "@/components/FeatureGuard";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import {
  featureToRef,
  useAuditLogStore,
  batchGetOrFetchProjects,
  pushNotification,
  useUserStore,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { type SearchAuditLogsParams } from "@/types";
import type { AuditLog } from "@/types/proto/v1/audit_log_service";
import { ExportFormat } from "@/types/proto/v1/common";
import { type SearchParams, extractProjectResourceName } from "@/utils";

interface LocalState {
  params: SearchParams;
}

const defaultSearchParams = () => {
  const params: SearchParams = {
    query: "",
    scopes: [],
  };
  return params;
};

const state = reactive<LocalState>({
  params: defaultSearchParams(),
});
const { t } = useI18n();
const auditLogStore = useAuditLogStore();
const auditLogPagedTable = ref<ComponentExposed<typeof PagedTable<AuditLog>>>();
const hasAuditLogFeature = featureToRef("bb.feature.audit-log");

const searchAuditLogs = computed((): SearchAuditLogsParams => {
  return {
    ...buildSearchAuditLogParams(state.params),
    order: "desc",
  };
});

const fetchAuditLog = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const { nextPageToken, auditLogs } = await auditLogStore.fetchAuditLogs({
    ...searchAuditLogs.value,
    pageToken,
    pageSize,
  });
  await batchGetOrFetchProjects(
    auditLogs.map((auditLog) => {
      const projectResourceId = extractProjectResourceName(auditLog.name);
      if (!projectResourceId) {
        return "";
      }
      return `${projectNamePrefix}${projectResourceId}`;
    })
  );
  await useUserStore().batchGetUsers(auditLogs.map((log) => log.user));
  return { nextPageToken, list: auditLogs };
};

watch(
  () => JSON.stringify(searchAuditLogs.value),
  () => auditLogPagedTable.value?.refresh()
);

const disableExportTip = computed(() => {
  if (
    !searchAuditLogs.value.createdTsAfter ||
    !searchAuditLogs.value.createdTsBefore
  ) {
    return t("audit-log.export-tooltip");
  }
  if (
    searchAuditLogs.value.createdTsBefore -
      searchAuditLogs.value.createdTsAfter >
    30 * 24 * 60 * 60 * 1000
  ) {
    return t("audit-log.export-tooltip");
  }
  return "";
});

const handleExport = async (
  options: ExportOption,
  callback: (content: BinaryLike | Blob, filename: string) => void
) => {
  let pageToken = "";
  let i = 0;

  while (i === 0 || pageToken !== "") {
    i++;
    const { content, nextPageToken } = await auditLogStore.exportAuditLogs({
      search: searchAuditLogs.value,
      format: options.format,
      pageSize: 10000,
    });
    pageToken = nextPageToken;
    callback(
      content,
      `audit-log${!pageToken && i === 1 ? "" : `.file${i}`}.${dayjs(new Date()).format("YYYY-MM-DDTHH-mm-ss")}`
    );
  }

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.success"),
    description: t("audit-log.export-finished"),
  });
};
</script>
