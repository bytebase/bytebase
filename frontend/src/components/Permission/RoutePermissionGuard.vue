<template>
  <slot v-if="missedPermissions.length === 0" />
  <NoPermissionPlaceholder
    v-else
    v-bind="$attrs"
    :resources="project ? [project.name] : []"
    :permissions="missedPermissions"
  />
</template>

<script lang="tsx" setup>
import { computed } from "vue";
import type { RouteRecordRaw } from "vue-router";
import { useRouter } from "vue-router";
import NoPermissionPlaceholder from "@/components/misc/NoPermissionPlaceholder.vue";
import { getFlattenRoutes } from "@/components/v2/Sidebar/utils.ts";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  project?: Project;
  routes: RouteRecordRaw[];
}>();

const router = useRouter();

const flattenRoutes = computed(() => {
  return getFlattenRoutes(props.routes);
});

const requiredPermissions = computed(() => {
  const routeConfig = flattenRoutes.value.find(
    (workspaceRoute) => workspaceRoute.name === router.currentRoute.value.name
  );
  return routeConfig?.permissions ?? [];
});

const missedPermissions = computed(() => {
  if (props.project) {
    return requiredPermissions.value.filter(
      (p) => !hasProjectPermissionV2(props.project!, p)
    );
  }
  return requiredPermissions.value.filter((p) => !hasWorkspacePermissionV2(p));
});
</script>