<template>
  <ComponentPermissionGuard :project="project" :permissions="requiredPermissions">
    <slot />
  </ComponentPermissionGuard>
</template>

<script lang="tsx" setup>
import { computed } from "vue";
import type { RouteRecordRaw } from "vue-router";
import { useRouter } from "vue-router";
import { getFlattenRoutes } from "@/components/v2/Sidebar/utils.ts";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import ComponentPermissionGuard from "./ComponentPermissionGuard.vue";

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
</script>