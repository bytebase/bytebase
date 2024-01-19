<template>
  <Suspense>
    <template #default>
      <ProvideInstanceContext :instance-id="instanceId">
        <router-view v-if="hasPermission" :instance-id="instanceId" />
        <NoPermissionPlaceholder v-else />
      </ProvideInstanceContext>
    </template>
    <template #fallback>
      <span>Loading instance...</span>
    </template>
  </Suspense>
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import { useRouter } from "vue-router";
import ProvideInstanceContext from "@/components/ProvideInstanceContext.vue";
import { useCurrentUserV1, useInstanceV1Store } from "@/store";
import { instanceNamePrefix } from "@/store/modules/v1/common";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  instanceId: string;
}>();

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

watchEffect(async () => {
  await useInstanceV1Store().getOrFetchInstanceByName(
    `${instanceNamePrefix}${props.instanceId}`
  );
});
</script>
