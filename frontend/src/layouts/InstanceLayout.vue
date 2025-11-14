<template>
  <Suspense>
    <template #default>
      <template v-if="hasPermission">
        <div
          v-if="!isValidInstanceName(instance.name)"
          class="flex items-center gap-x-2 m-4"
        >
          <BBSpin :size="20" />
          Loading instance...
        </div>
        <router-view v-else :instance-id="instanceId" />
      </template>
      <NoPermissionPlaceholder v-else class="py-6" />
    </template>
    <template #fallback>
      <span>Loading instance...</span>
    </template>
  </Suspense>
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import { useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import NoPermissionPlaceholder from "@/components/misc/NoPermissionPlaceholder.vue";
import { useInstanceV1Store } from "@/store";
import { instanceNamePrefix } from "@/store/modules/v1/common";
import { isValidInstanceName } from "@/types";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  instanceId: string;
}>();

const router = useRouter();
const instanceStore = useInstanceV1Store();

const requiredPermissions = computed(() => {
  const getPermissionListFunc =
    router.currentRoute.value.meta.requiredPermissionList;
  return getPermissionListFunc ? getPermissionListFunc() : [];
});

const hasPermission = computed(() => {
  return requiredPermissions.value.every((permission) =>
    hasWorkspacePermissionV2(permission)
  );
});

watchEffect(async () => {
  if (!hasPermission.value) {
    return;
  }

  await instanceStore.getOrFetchInstanceByName(
    `${instanceNamePrefix}${props.instanceId}`
  );
});

const instance = computed(() =>
  instanceStore.getInstanceByName(`${instanceNamePrefix}${props.instanceId}`)
);
</script>
