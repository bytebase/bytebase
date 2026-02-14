<template>
  <NTooltip v-if="ready">
    <template #trigger>
      <ShieldCheckIcon v-if="configured" class="w-4 h-4 text-success" />
      <TriangleAlertIcon v-else class="w-4 h-4 text-warning" />
    </template>
    <div class="flex flex-col gap-y-1">
      <span>{{ tooltipText }}</span>
      <router-link
        class="normal-link"
        :to="{ name: WORKSPACE_ROUTE_CUSTOM_APPROVAL }"
      >
        {{
          $t("project.settings.issue-related.view-approval-flow")
        }}
      </router-link>
    </div>
  </NTooltip>
</template>

<script setup lang="ts">
import { ShieldCheckIcon, TriangleAlertIcon } from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { computed, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { WORKSPACE_ROUTE_CUSTOM_APPROVAL } from "@/router/dashboard/workspaceRoutes";
import { useWorkspaceApprovalSettingStore } from "@/store";
import { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  source: WorkspaceApprovalSetting_Rule_Source;
}>();

const { t } = useI18n();
const approvalSettingStore = useWorkspaceApprovalSettingStore();
const ready = ref(false);

watchEffect(async () => {
  if (!hasWorkspacePermissionV2("bb.settings.get")) {
    return;
  }
  await approvalSettingStore.fetchConfig();
  ready.value = true;
});

const status = computed((): "source" | "fallback" | "none" => {
  if (approvalSettingStore.getRulesBySource(props.source).length > 0) {
    return "source";
  }
  if (
    approvalSettingStore.getRulesBySource(
      WorkspaceApprovalSetting_Rule_Source.SOURCE_UNSPECIFIED
    ).length > 0
  ) {
    return "fallback";
  }
  return "none";
});

const configured = computed(() => status.value !== "none");

const tooltipText = computed(() => {
  switch (status.value) {
    case "source":
      return t("project.settings.issue-related.approval-flow-configured");
    case "fallback":
      return t("project.settings.issue-related.approval-flow-fallback");
    default:
      return t("project.settings.issue-related.approval-flow-not-configured");
  }
});
</script>
