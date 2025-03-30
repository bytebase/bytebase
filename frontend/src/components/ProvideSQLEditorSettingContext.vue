<template>
  <router-view />
</template>

<script lang="ts" setup>
import { onMounted } from "vue";
import { useGroupStore, useRoleStore } from "@/store";

onMounted(async () => {
  await Promise.all([
    useRoleStore().fetchRoleList(),
    useGroupStore().fetchGroupList(),
  ]);
});
</script>
