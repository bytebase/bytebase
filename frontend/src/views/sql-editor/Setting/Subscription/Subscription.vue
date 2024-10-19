<template>
  <div
    class="flex flex-col py-4 px-4 gap-4 divide-block-border"
    v-bind="$attrs"
  >
    <SettingWorkspaceSubscription :allow-edit="allowEdit" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { hasWorkspacePermissionV2 } from "@/utils";
import SettingWorkspaceSubscription from "@/views/SettingWorkspaceSubscription.vue";

const allowEdit = computed(() => {
  return hasWorkspacePermissionV2("bb.settings.set");
});
</script>
