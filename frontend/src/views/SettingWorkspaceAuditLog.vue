<template>
  <div class="w-full flex flex-col gap-y-4 pb-6">
    <FeatureAttention :feature="PlanFeature.FEATURE_AUDIT_LOG" />
    <AuditLogSearch v-model:params="state.params">
      <template #searchbox-suffix>
        <DataExportButton
          size="medium"
          :support-formats="[
            ExportFormat.CSV,
            ExportFormat.JSON,
            ExportFormat.XLSX,
          ]"
          :tooltip="disableExportTip"
          :view-mode="'DROPDOWN'"
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
    <NEmpty class="py-12 border rounded-sm" v-else />
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { NEmpty } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useI18n } from "vue-i18n";
import AuditLogDataTable from "@/components/AuditLog/AuditLogDataTable.vue";
import AuditLogSearch from "@/components/AuditLog/AuditLogSearch";
import { buildSearchAuditLogParams } from "@/components/AuditLog/AuditLogSearch/utils";
import type {
  DownloadContent,
  ExportOption,
} from "@/components/DataExportButton.vue";
import DataExportButton from "@/components/DataExportButton.vue";
import { FeatureAttention } from "@/components/FeatureGuard";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { featureToRef, useAuditLogStore } from "@/store";
import { type SearchAuditLogsParams } from "@/types";
import type { AuditLog } from "@/types/proto-es/v1/audit_log_service_pb";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { type SearchParams } from "@/utils";

interface LocalState {
  params: SearchParams;
}

const defaultSearchParams = () => {
  const to = dayjs().endOf("day");
  const from = to.add(-30, "day");
  const params: SearchParams = {
    query: "",
    scopes: [
      {
        id: "created",
        value: `${from.valueOf()},${to.valueOf()}`,
      },
    ],
  };
  return params;
};

const state = reactive<LocalState>({
  params: defaultSearchParams(),
});
const { t } = useI18n();
const auditLogStore = useAuditLogStore();
const auditLogPagedTable = ref<ComponentExposed<typeof PagedTable<AuditLog>>>();
const hasAuditLogFeature = featureToRef(PlanFeature.FEATURE_AUDIT_LOG);

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
        search: searchAuditLogs.value,
        format: options.format,
        pageSize: 5000, // The maximum page size is 5000
        pageToken,
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
</script>
