<template>
  <div class="w-full flex flex-col gap-y-4 pb-6">
    <FeatureAttention :feature="PlanFeature.FEATURE_AUDIT_LOG" />
    <AuditLogSearch v-model:params="state.params">
      <template #searchbox-suffix>
        <DataExportButton
          v-if="hasWorkspacePermissionV2('bb.auditLogs.export')"
          size="medium"
          :support-formats="[
            ExportFormat.CSV,
            ExportFormat.JSON,
            ExportFormat.XLSX,
          ]"
          :tooltip="disableExportTip"
          :view-mode="'DROPDOWN'"
          :disabled="!hasAuditLogFeature || !!disableExportTip"
          @export="(params) => pagedAuditLogDataTableRef?.handleExport(params)"
        />
      </template>
    </AuditLogSearch>

    <PagedAuditLogDataTable
      v-if="hasAuditLogFeature"
      ref="pagedAuditLogDataTableRef"
      :parent="parent"
      :filter="searchAuditLogs"
    />
    <NEmpty class="py-12 border rounded-sm" v-else />
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { NEmpty } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import AuditLogSearch from "@/components/AuditLog/AuditLogSearch";
import { buildAuditLogFilter } from "@/components/AuditLog/AuditLogSearch/utils";
import PagedAuditLogDataTable from "@/components/AuditLog/PagedAuditLogDataTable.vue";
import DataExportButton from "@/components/DataExportButton.vue";
import { FeatureAttention } from "@/components/FeatureGuard";
import { featureToRef } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { type AuditLogFilter } from "@/types";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2, type SearchParams } from "@/utils";

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
const hasAuditLogFeature = featureToRef(PlanFeature.FEATURE_AUDIT_LOG);
const pagedAuditLogDataTableRef =
  ref<InstanceType<typeof PagedAuditLogDataTable>>();

const parent = computed(() => `${projectNamePrefix}-`);

const searchAuditLogs = computed((): AuditLogFilter => {
  return buildAuditLogFilter(state.params);
});

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
</script>
