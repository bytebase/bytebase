<template>
  <div class="w-full flex flex-col gap-4 py-4 px-2 overflow-y-auto">
    <SettingWorkspaceDataMasking :embedded="true" />
  </div>
</template>

<script setup lang="ts">
import SettingWorkspaceDataMasking from "@/views/SettingWorkspaceDataMasking.vue";
</script>
