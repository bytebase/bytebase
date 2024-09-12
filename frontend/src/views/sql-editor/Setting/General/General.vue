<template>
  <div
    class="w-full h-full flex flex-col py-4 px-4 divide-y divide-block-border overflow-y-auto"
  >
    <SettingWorkspaceGeneral :allow-edit="allowEdit" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { hasWorkspacePermissionV2 } from "@/utils";
import SettingWorkspaceGeneral from "@/views/SettingWorkspaceGeneral.vue";

const allowEdit = computed((): boolean => {
  return hasWorkspacePermissionV2("bb.settings.set");
});
</script>
