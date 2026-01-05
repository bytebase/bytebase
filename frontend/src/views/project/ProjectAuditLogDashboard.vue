<template>
  <div class="w-full flex flex-col gap-y-4">
    <FeatureAttention :feature="PlanFeature.FEATURE_AUDIT_LOG" />
    <AuditLogSearch v-model:params="state.params">
      <template #searchbox-suffix>
        <DataExportButton
          v-if="hasProjectPermissionV2(project, 'bb.auditLogs.export')"
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
      :parent="projectName"
      :filter="searchAuditLogs"
    />
    <NEmpty class="py-12 border rounded-sm" v-else />
  </div>
</template>

<script setup lang="ts">
import dayjs from "dayjs";
import { NEmpty } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import AuditLogSearch from "@/components/AuditLog/AuditLogSearch";
import { buildAuditLogFilter } from "@/components/AuditLog/AuditLogSearch/utils";
import PagedAuditLogDataTable from "@/components/AuditLog/PagedAuditLogDataTable.vue";
import DataExportButton from "@/components/DataExportButton.vue";
import { FeatureAttention } from "@/components/FeatureGuard";
import { featureToRef, useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { AuditLogFilter } from "@/types";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { SearchParams, SearchScope } from "@/utils";
import { hasProjectPermissionV2 } from "@/utils";

interface LocalState {
  params: SearchParams;
}

const props = defineProps<{
  projectId: string;
}>();

const readonlyScopes = computed((): SearchScope[] => {
  return [{ id: "project", value: props.projectId, readonly: true }];
});

const projectName = computed(() => `${projectNamePrefix}${props.projectId}`);

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

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const { t } = useI18n();
const hasAuditLogFeature = featureToRef(PlanFeature.FEATURE_AUDIT_LOG);
const pagedAuditLogDataTableRef =
  ref<InstanceType<typeof PagedAuditLogDataTable>>();

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
