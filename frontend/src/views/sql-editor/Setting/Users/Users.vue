<template>
  <div
    class="w-full flex flex-col py-4 px-4 gap-4 divide-block-border overflow-y-auto"
    v-bind="$attrs"
  >
    <SettingWorkspaceUsers />
  </div>
</template>

<script lang="ts" setup>
import SettingWorkspaceUsers from "@/views/SettingWorkspaceUsers.vue";
</script>
