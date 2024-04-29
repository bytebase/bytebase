<template>
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
</template>

<script setup lang="ts">
import { computed } from "vue";
import AuditLogDataTable from "@/components/AuditLog/AuditLogDataTable.vue";
import PagedAuditLogTable from "@/components/PagedAuditLogTable.vue";
import { featureToRef, useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { SearchAuditLogsParams } from "@/types";

const props = defineProps<{
  projectId: string;
}>();

const hasAuditLogFeature = featureToRef("bb.feature.audit-log");

const projectV1Store = useProjectV1Store();

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

const searchAuditLogs = computed((): SearchAuditLogsParams => {
  return {
    parent: project.value.name,
    order: "desc",
  };
});
</script>
