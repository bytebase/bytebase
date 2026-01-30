<template>
  <RequiredBasicPermissionAlert
    v-if="missedBasicPermission.length > 0"
    v-bind="$attrs"
    :permissions="missedBasicPermission"
    :title="$t('common.workspace-initialize-error')"
  />
  <template v-else>
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
          <PermissionGuardWrapper
            v-slot="slotProps"
            :project="project"
            :permissions="['bb.issues.create', 'bb.roles.list']"
          >
            <NButton
              type="primary"
              :disabled="slotProps.disabled || !hasRequestRoleFeature"
              @click="showRequestRolePanel = true"
            >
            <template #icon>
              <ShieldUserIcon v-if="hasRequestRoleFeature" class="w-4 h-4" />
              <FeatureBadge v-else :clickable="false" :feature="PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW" />
            </template>
            {{ $t("issue.title.request-role") }}
          </NButton>
          </PermissionGuardWrapper>
        </div>
      </template>
    </NoPermissionPlaceholder>
  </template>

  <GrantRequestPanel
    v-if="showRequestRolePanel && project"
    :project-name="project.name"
    :required-permissions="missedPermissions"
    @close="showRequestRolePanel = false"
  />
</template>


<script lang="tsx" setup>
import { ShieldUserIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import { FeatureBadge } from "@/components/FeatureGuard";
import GrantRequestPanel from "@/components/GrantRequestPanel";
import NoPermissionPlaceholder from "@/components/Permission/NoPermissionPlaceholder.vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import RequiredBasicPermissionAlert from "@/components/Role/Setting/components/RequiredBasicPermissionAlert.vue";
import { hasFeature } from "@/store";
import { BASIC_WORKSPACE_PERMISSIONS, type Permission } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  project?: Project;
  permissions: Permission[];
}>();

const showRequestRolePanel = ref(false);
const router = useRouter();
const hasRequestRoleFeature = computed(() =>
  hasFeature(PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW)
);

const allowRequestRole = computed(() => {
  return props.project?.allowRequestRole;
});

const requestPath = computed(() => {
  return router.currentRoute.value.fullPath;
});

const missedPermissions = computed(() => {
  if (props.project) {
    return props.permissions.filter(
      (p) => !hasProjectPermissionV2(props.project!, p)
    );
  }
  return props.permissions.filter((p) => !hasWorkspacePermissionV2(p));
});

const missedBasicPermission = computed(() => {
  return BASIC_WORKSPACE_PERMISSIONS.filter(
    (p) => !hasWorkspacePermissionV2(p)
  );
});
</script>