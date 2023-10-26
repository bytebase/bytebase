<template>
  <div class="p-4">
    <AnomalyCenterDashboard :selected-tab="selectedTab" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import AnomalyCenterDashboard from "@/components/AnomalyCenter/AnomalyCenterDashboard.vue";
import { useCurrentUserV1 } from "@/store";
import { hasWorkspacePermissionV1 } from "@/utils";

const currentUserV1 = useCurrentUserV1();
const selectedTab = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-instance",
    currentUserV1.value.userRole
  )
    ? 1
    : 0;
});
</script>
