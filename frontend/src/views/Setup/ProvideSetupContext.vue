<template>
  <slot v-if="ready" />
  <MaskSpinner v-else class="!bg-white" />
</template>

<script lang="ts" setup>
import { ref, onMounted, watch } from "vue";
import {
  useRouter,
  type RouteLocationNormalizedLoadedGeneric,
} from "vue-router";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { usePolicyV1Store, useRoleStore, useSettingV1Store } from "@/store";
import { PolicyResourceType } from "@/types/proto/v1/org_policy_service";
import { hasWorkspacePermissionV2 } from "@/utils";

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

  // Prepare roles, workspace policies and settings first.
  await Promise.all([
    useRoleStore().fetchRoleList(),
    usePolicyV1Store().fetchPolicies({
      resourceType: PolicyResourceType.WORKSPACE,
    }),
    useSettingV1Store().fetchSettingList(),
  ]);

  watch(
    router.currentRoute,
    (route) => {
      if (!checkPermissions(route)) {
        ready.value = false;
        router.push({
          name: "error.403",
        });
        return;
      }
      ready.value = true;
    },
    { immediate: true, flush: "post" }
  );
});
</script>
