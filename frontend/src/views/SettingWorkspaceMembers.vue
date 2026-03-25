<template>
  <div class="px-4 pt-2 pb-4 w-full">
    <WorkspaceMembers
      :tab-type="'line'"
    />
  </div>
</template>

<script setup lang="ts">
import WorkspaceMembers from "@/components/Member/Views/WorkspaceMembers.vue";
</script>
