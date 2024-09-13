<template>
  <div class="w-full flex flex-col gap-4 py-4 px-2 overflow-y-auto">
    <SettingWorkspaceDataAccessControl :embedded="true" />
  </div>
</template>

<script setup lang="ts">
import SettingWorkspaceDataAccessControl from "@/views/SettingWorkspaceDataAccessControl.vue";
</script>
