<template>
  <div class="w-full space-y-4">
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
          :disabled="!hasAuditLogFeature"
          @export="handleExport"
        />
      </template>
    </AuditLogSearch>
    <PagedAuditLogTable
      v-if="hasAuditLogFeature"
      :search-audit-logs="searchAuditLogs"
      session-key="bb.page-audit-log-table.settings-audit-log-v1-table"
      :page-size="10"
    >
      <template #table="{ list }">
        <AuditLogDataTable key="audit-log-table" :audit-log-list="list" />
      </template>
    </PagedAuditLogTable>
    <template v-else>
      <AuditLogDataTable key="audit-log-table" :audit-log-list="[]" />
    </template>
  </div>
</template>

<script lang="ts" setup>
import type { BinaryLike } from "node:crypto";
import { reactive, computed } from "vue";
import AuditLogDataTable from "@/components/AuditLog/AuditLogDataTable.vue";
import AuditLogSearch from "@/components/AuditLog/AuditLogSearch";
import { buildSearchAuditLogParams } from "@/components/AuditLog/AuditLogSearch/utils";
import type { ExportOption } from "@/components/DataExportButton.vue";
import DataExportButton from "@/components/DataExportButton.vue";
import { FeatureAttention } from "@/components/FeatureGuard";
import PagedAuditLogTable from "@/components/PagedAuditLogTable.vue";
import { featureToRef, useAuditLogStore } from "@/store";
import type { SearchAuditLogsParams } from "@/types";
import { ExportFormat } from "@/types/proto/v1/common";
import { type SearchParams } from "@/utils";

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

const auditLogStore = useAuditLogStore();

const hasAuditLogFeature = featureToRef("bb.feature.audit-log");

const searchAuditLogs = computed((): SearchAuditLogsParams => {
  return {
    ...buildSearchAuditLogParams(state.params),
    order: "desc",
  };
});

const handleExport = async (
  options: ExportOption,
  callback: (content: BinaryLike | Blob, options: ExportOption) => void
) => {
  const content = await auditLogStore.exportAuditLogs(
    searchAuditLogs.value,
    options.format
  );
  callback(content, options);
};
</script>
