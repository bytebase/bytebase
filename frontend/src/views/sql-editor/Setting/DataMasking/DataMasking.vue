<template>
  <div class="flex flex-col gap-4 py-4 px-4">
    <SettingWorkspaceDataMasking :embedded="true" />
  </div>
</template>

<script setup lang="ts">
import SettingWorkspaceDataMasking from "@/views/SettingWorkspaceDataMasking.vue";
</script>
