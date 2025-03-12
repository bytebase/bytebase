<template>
  <div class="px-4 h-full">
    <router-view v-if="hasPermission" :allow-edit="allowEdit" v-bind="$attrs" />
    <NoPermissionPlaceholder v-else class="py-6" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import NoPermissionPlaceholder from "@/components/misc/NoPermissionPlaceholder.vue";
import { hasWorkspacePermissionV2 } from "@/utils";

const router = useRouter();

const requiredPermissions = computed(() => {
  const getPermissionListFunc =
    router.currentRoute.value.meta.requiredWorkspacePermissionList;
  return getPermissionListFunc ? getPermissionListFunc() : [];
});

const hasPermission = computed(() => {
  return requiredPermissions.value.every((permission) =>
    hasWorkspacePermissionV2(permission)
  );
});

const allowEdit = computed((): boolean => {
  return hasWorkspacePermissionV2("bb.settings.set");
});
</script>
