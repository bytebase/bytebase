<template>
  <slot />
</template>

<script setup lang="ts">
import { onMounted } from "vue";
import { useRouter } from "vue-router";
import { WORKSPACE_ROUTE_MY_ISSUES } from "@/router/dashboard/workspaceRoutes";

const router = useRouter();

onMounted(() => {
  router.replace({
    name: WORKSPACE_ROUTE_MY_ISSUES,
  });
});
</script>
