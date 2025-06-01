<template>
  <div class="w-full space-y-4">
    <FeatureAttention feature="bb.feature.audit-log" />
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
      :session-key="`bb.page-audit-log-table.settings-audit-log-v1-table.${projectId}`"
      :fetch-list="fetchAuditLog"
    >
      <template #table="{ list, loading }">
        <AuditLogDataTable
          :key="`audit-log-table.${projectId}`"
          :audit-log-list="list"
          :show-project="false"
          :loading="loading"
        />
      </template>
    </PagedTable>
    <NEmpty class="py-12 border rounded" v-else />
  </div>
</template>

<script setup lang="ts">
import dayjs from "dayjs";
import { NEmpty } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
// https://github.com/vuejs/language-tools/issues/3206
import type { ComponentExposed } from "vue-component-type-helpers";
import { useI18n } from "vue-i18n";
import AuditLogDataTable from "@/components/AuditLog/AuditLogDataTable.vue";
import AuditLogSearch from "@/components/AuditLog/AuditLogSearch";
import { buildSearchAuditLogParams } from "@/components/AuditLog/AuditLogSearch/utils";
import type {
  ExportOption,
  DownloadContent,
} from "@/components/DataExportButton.vue";
import DataExportButton from "@/components/DataExportButton.vue";
import { FeatureAttention } from "@/components/FeatureGuard";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { featureToRef, useAuditLogStore, useUserStore } from "@/store";
import { type SearchAuditLogsParams } from "@/types";
import type { AuditLog } from "@/types/proto/v1/audit_log_service";
import { ExportFormat } from "@/types/proto/v1/common";
import type { SearchParams, SearchScope } from "@/utils";

interface LocalState {
  params: SearchParams;
}

const props = defineProps<{
  projectId: string;
}>();

const readonlyScopes = computed((): SearchScope[] => {
  return [{ id: "project", value: props.projectId, readonly: true }];
});

const defaultSearchParams = () => {
  const to = dayjs().endOf("day");
  const from = to.add(-30, "day");
  const params: SearchParams = {
    query: "",
    scopes: [
      ...readonlyScopes.value,
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

watch(
  () => props.projectId,
  () => (state.params = defaultSearchParams())
);

const { t } = useI18n();
const hasAuditLogFeature = featureToRef("bb.feature.audit-log");
const auditLogPagedTable = ref<ComponentExposed<typeof PagedTable<AuditLog>>>();
const auditLogStore = useAuditLogStore();

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

const handleExport = async ({
  options,
  resolve,
  reject,
}: {
  options: ExportOption;
  reject: (reason?: any) => void;
  resolve: (content: DownloadContent) => void;
}) => {
  let pageToken = "";
  let i = 0;
  const contents: DownloadContent = [];

  try {
    while (i === 0 || pageToken !== "") {
      i++;
      const { content, nextPageToken } = await auditLogStore.exportAuditLogs({
        search: searchAuditLogs.value,
        format: options.format,
        pageSize: 10000,
        pageToken,
      });
      pageToken = nextPageToken;
      contents.push({
        content,
        filename: `audit-log.file${i}.${dayjs(new Date()).format("YYYY-MM-DDTHH-mm-ss")}`,
      });
    }
    resolve(contents);
  } catch (err) {
    reject(err);
  }
};
</script>
