<template>
  <div class="px-4 pb-6">
    <AnomalyCenterDashboard :selected-tab="selectedTab" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import AnomalyCenterDashboard, {
  AnomalyTabId,
} from "@/components/AnomalyCenter/AnomalyCenterDashboard.vue";
import { useCurrentUserV1 } from "@/store";
import { hasWorkspacePermissionV2 } from "@/utils";

const currentUserV1 = useCurrentUserV1();
const selectedTab = computed((): AnomalyTabId => {
  return hasWorkspacePermissionV2(currentUserV1.value, "bb.instances.update")
    ? "instance"
    : "database";
});
</script>
