<template>
  <div class="w-full space-y-4">
    <FeatureAttention feature="bb.feature.audit-log" />
    <div class="flex justify-end items-center space-x-2">
      <div class="w-72">
        <UserSelect
          v-model:user="state.userUid"
          :multiple="false"
          :include-all="true"
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
        <AuditLogDataTable :audit-log-list="list" />
      </template>
    </PagedAuditLogTable>
    <template v-else>
      <AuditLogDataTable :audit-log-list="[]" />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { reactive, computed } from "vue";
import AuditLogDataTable from "@/components/AuditLog/AuditLogDataTable.vue";
import PagedAuditLogTable from "@/components/PagedAuditLogTable.vue";
import { featureToRef, useUserStore } from "@/store";
import type { SearchAuditLogsParams } from "@/types";
import { UNKNOWN_ID } from "@/types";

const state = reactive({
  userUid: String(UNKNOWN_ID),
});

const userStore = useUserStore();

const hasAuditLogFeature = featureToRef("bb.feature.audit-log");

const selectedUserEmail = computed((): string => {
  const selected = userStore.getUserById(state.userUid);
  return selected?.email ?? "";
});

const searchAuditLogs = computed((): SearchAuditLogsParams => {
  return {
    creatorEmail: selectedUserEmail.value,
    order: "desc",
  };
});
</script>
