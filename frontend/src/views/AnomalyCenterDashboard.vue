<template>
  <div class="px-4 py-6">
    <AnomalyCenterDashboard :selected-tab="selectedTab" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import AnomalyCenterDashboard, {
  AnomalyTabId,
} from "@/components/AnomalyCenter/AnomalyCenterDashboard.vue";
import { useCurrentUserV1 } from "@/store";
import { hasWorkspacePermissionV1 } from "@/utils";

const currentUserV1 = useCurrentUserV1();
const selectedTab = computed((): AnomalyTabId => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-instance",
    currentUserV1.value.userRole
  )
    ? "instance"
    : "database";
});
</script>
