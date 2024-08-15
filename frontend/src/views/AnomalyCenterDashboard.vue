<template>
  <div class="px-4">
    <AnomalyCenterDashboard :selected-tab="selectedTab" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type { AnomalyTabId } from "@/components/AnomalyCenter/AnomalyCenterDashboard.vue";
import AnomalyCenterDashboard from "@/components/AnomalyCenter/AnomalyCenterDashboard.vue";
import { hasWorkspacePermissionV2 } from "@/utils";

const selectedTab = computed((): AnomalyTabId => {
  return hasWorkspacePermissionV2("bb.instances.update")
    ? "instance"
    : "database";
});
</script>
