<template>
  <RoutePermissionGuard :routes="setupRoutes">
    <AdminSetup v-if="ready" />
    <MaskSpinner v-else class="bg-white!" />
    <AuthFooter />
  </RoutePermissionGuard>
</template>

<script lang="ts" setup>
import { onMounted, ref } from "vue";
import {
  useRouter,
} from "vue-router";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { useRoleStore, useWorkspaceV1Store } from "@/store";
import AuthFooter from "@/views/auth/AuthFooter.vue";
import AdminSetup from "./AdminSetup.vue";
import RoutePermissionGuard from "@/components/Permission/RoutePermissionGuard.vue";
import setupRoutes from "@/router/setup"

const router = useRouter();
const ready = ref<boolean>(false);

onMounted(async () => {
  await router.isReady();
  // Prepare roles and workspace IAM policy.
  await Promise.all([
    useRoleStore().fetchRoleList(),
    useWorkspaceV1Store().fetchIamPolicy(),
  ]);
});
</script>
