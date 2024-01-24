<template>
  <div class="px-4">
    <router-view v-if="hasPermission" :allow-edit="allowEdit" v-bind="$attrs" />
    <NoPermissionPlaceholder v-else />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
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

const allowEdit = computed((): boolean => {
  return hasWorkspacePermissionV2(currentUser.value, "bb.settings.set");
});
</script>
