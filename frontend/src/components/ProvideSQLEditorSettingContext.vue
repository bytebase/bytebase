<template>
  <router-view />
</template>

<script lang="ts" setup>
import { onMounted } from "vue";
import { useGroupStore, useRoleStore, useUserStore } from "@/store";

onMounted(() => {
  useUserStore().fetchUserList();
  useRoleStore().fetchRoleList();
  useGroupStore().fetchGroupList();
});
</script>
