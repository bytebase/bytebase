<template>
  <AdminSetup v-if="ready" />
  <MaskSpinner v-else class="bg-white!" />
  <AuthFooter />
</template>

<script lang="ts" setup>
import { onMounted, ref, watch } from "vue";
import {
  type RouteLocationNormalizedLoadedGeneric,
  useRouter,
} from "vue-router";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { WORKSPACE_ROUTE_403 } from "@/router/dashboard/workspaceRoutes";
import { useRoleStore, useWorkspaceV1Store } from "@/store";
import { hasWorkspacePermissionV2 } from "@/utils";
import AuthFooter from "@/views/auth/AuthFooter.vue";
import AdminSetup from "./AdminSetup.vue";

const router = useRouter();
const ready = ref<boolean>(false);

const checkPermissions = (route: RouteLocationNormalizedLoadedGeneric) => {
  if (!route.meta.requiredPermissionList) {
    return true;
  }
  const requiredPermissionList = route.meta.requiredPermissionList();
  if (!requiredPermissionList.every(hasWorkspacePermissionV2)) {
    return false;
  }
  return true;
};

onMounted(async () => {
  await router.isReady();

  // Prepare roles and workspace IAM policy.
  await Promise.all([
    useRoleStore().fetchRoleList(),
    useWorkspaceV1Store().fetchIamPolicy(),
  ]);

  watch(
    router.currentRoute,
    (route) => {
      if (!checkPermissions(route)) {
        ready.value = false;
        router.push({
          name: WORKSPACE_ROUTE_403,
        });
        return;
      }
      ready.value = true;
    },
    { immediate: true, flush: "post" }
  );
});
</script>
