<template>
  <router-view />
</template>

<script lang="ts" setup>
import { onMounted } from "vue";
import { useRoleStore } from "@/store";

onMounted(async () => {
  await useRoleStore().fetchRoleList();
});
</script>
