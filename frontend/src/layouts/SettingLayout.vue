<template>
  <div class="p-6">
    <router-view />
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, watch } from "vue";
import { useRouter } from "vue-router";
import { useCurrentUserV1 } from "@/store";
import { hasWorkspacePermissionV2 } from "@/utils";

const router = useRouter();
const currentUser = useCurrentUserV1();

const requiredPermissions = computed(() => {
  const getPermissionListFunc =
    router.currentRoute.value.meta.requiredWorkspacePermissionList;
  return getPermissionListFunc ? getPermissionListFunc() : [];
});

const hasPermission = computed(() => {
  return requiredPermissions.value.every((permission) =>
    hasWorkspacePermissionV2(currentUser.value, permission)
  );
});

onMounted(() => {});

watch(
  () => hasPermission.value,
  (hasPermission) => {
    if (!hasPermission) {
      router.push({
        name: "error.403",
        replace: false,
      });
    }
  },
  {
    immediate: true,
  }
);
</script>
