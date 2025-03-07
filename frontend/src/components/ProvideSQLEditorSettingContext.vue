<template>
  <router-view />
</template>

<script lang="ts" setup>
import { onMounted } from "vue";
import { useGroupStore, useRoleStore, useUserStore } from "@/store";

onMounted(async () => {
  await Promise.all([
    // TODO(ed): support pagination.
    useUserStore().fetchUserList({ pageSize: 100000 }),
    useRoleStore().fetchRoleList(),
    useGroupStore().fetchGroupList(),
  ]);
});
</script>
