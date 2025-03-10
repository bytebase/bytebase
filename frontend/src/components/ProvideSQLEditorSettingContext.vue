<template>
  <router-view />
</template>

<script lang="ts" setup>
import { onMounted } from "vue";
import { useGroupStore, useRoleStore, useUserStore } from "@/store";

onMounted(async () => {
  await Promise.all([
    useUserStore().refreshUserStat(),
    useRoleStore().fetchRoleList(),
    useGroupStore().fetchGroupList(),
  ]);
});
</script>
