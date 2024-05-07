<template>
  <div class="w-full space-y-4">
    <FeatureAttention feature="bb.feature.audit-log" />
    <div class="flex justify-between items-center space-x-2">
      <div>
        <div class="w-72">
          <UserSelect
            v-model:user="state.userUid"
            :multiple="false"
            :include-all="true"
          />
        </div>
      </div>
      <div>
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
      </div>
    </div>

    <PagedAuditLogTable
      v-if="hasAuditLogFeature"
      :search-audit-logs="searchAuditLogs"
      session-key="bb.page-audit-log-table.settings-audit-log-v1-table"
      :page-size="10"
    >
      <template #table="{ list }">
        <AuditLogDataTable :audit-log-list="list" :show-project="false" />
      </template>
    </PagedAuditLogTable>
    <template v-else>
      <AuditLogDataTable :audit-log-list="[]" :show-project="false" />
    </template>
  </div>
</template>

<script setup lang="ts">
import type { BinaryLike } from "node:crypto";
import { computed, reactive } from "vue";
import AuditLogDataTable from "@/components/AuditLog/AuditLogDataTable.vue";
import type { ExportOption } from "@/components/DataExportButton.vue";
import PagedAuditLogTable from "@/components/PagedAuditLogTable.vue";
import {
  featureToRef,
  useAuditLogStore,
  useProjectV1Store,
  useUserStore,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { UNKNOWN_ID, type SearchAuditLogsParams } from "@/types";
import { ExportFormat } from "@/types/proto/v1/common";

const props = defineProps<{
  projectId: string;
}>();

const state = reactive({
  userUid: String(UNKNOWN_ID),
});

const hasAuditLogFeature = featureToRef("bb.feature.audit-log");

const userStore = useUserStore();
const projectV1Store = useProjectV1Store();
const auditLogStore = useAuditLogStore();

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

const selectedUserEmail = computed((): string => {
  const selected = userStore.getUserById(state.userUid);
  return selected?.email ?? "";
});

const searchAuditLogs = computed((): SearchAuditLogsParams => {
  return {
    parent: project.value.name,
    creatorEmail: selectedUserEmail.value,
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
