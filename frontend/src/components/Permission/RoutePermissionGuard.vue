<template>
  <slot v-if="missedPermissions.length === 0" />
  <NoPermissionPlaceholder
    v-else
    v-bind="$attrs"
    :path="requestPath"
    :resources="project ? [project.name] : []"
    :permissions="missedPermissions"
  >
    <template #action>
      <div v-if="allowRequestRole" class="mt-2">
        <NButton
          type="primary"
          @click="showRequestRolePanel = true"
        >
          <template #icon>
            <heroicons-outline:user-add class="w-4 h-4" />
          </template>
          {{ $t("issue.title.request-role") }}
        </NButton>
      </div>
    </template>
  </NoPermissionPlaceholder>

  <GrantRequestPanel
    v-if="showRequestRolePanel && project"
    :project-name="project.name"
    @close="showRequestRolePanel = false"
  />
</template>

<script lang="tsx" setup>
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import type { RouteRecordRaw } from "vue-router";
import { useRouter } from "vue-router";
import GrantRequestPanel from "@/components/GrantRequestPanel";
import NoPermissionPlaceholder from "@/components/Permission/NoPermissionPlaceholder.vue";
import { getFlattenRoutes } from "@/components/v2/Sidebar/utils.ts";
import { hasFeature } from "@/store";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  project?: Project;
  routes: RouteRecordRaw[];
}>();

const showRequestRolePanel = ref(false);
const router = useRouter();
const allowRequestRole = computed(() => {
  return (
    props.project?.allowRequestRole &&
    hasFeature(PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW) &&
    hasProjectPermissionV2(props.project, "bb.issues.create") &&
    hasWorkspacePermissionV2("bb.roles.list")
  );
});

const flattenRoutes = computed(() => {
  return getFlattenRoutes(props.routes);
});

const requestPath = computed(() => {
  return router.currentRoute.value.fullPath;
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