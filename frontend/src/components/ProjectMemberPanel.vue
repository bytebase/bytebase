<template>
  <div class="">
    <FeatureAttention
      v-if="!hasRBACFeature"
      custom-class="my-5"
      feature="bb.feature.rbac"
      :description="$t('subscription.features.bb-feature-rbac.desc')"
    />
    <span class="text-lg font-medium leading-7 text-main">
      {{ $t("project.settings.manage-member") }}
    </span>

    <div v-if="allowAdmin" class="my-4 w-full flex gap-x-2">
      <div class="w-[18rem] shrink-0">
        <PrincipalSelect
          v-model:principal="state.principalId"
          :include-all="false"
          :filter="filterNonMemberUsers"
          :disabled="state.adding"
          style="width: 100%"
        />
      </div>
      <div v-if="hasRBACFeature" class="flex-1 overflow-hidden">
        <ProjectRolesSelect
          v-model:role-list="state.roleList"
          :disabled="state.adding"
        />
      </div>
      <NButton
        type="primary"
        :disabled="!isValid"
        :loading="state.adding"
        @click="addMember"
      >
        <template #icon>
          <heroicons-outline:user-add class="w-4 h-4" />
        </template>
        {{ $t("project.settings.add-member") }}
      </NButton>
    </div>
    <ProjectMemberTable
      :iam-policy="iamPolicy"
      :project="project"
      :ready="ready"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive } from "vue";
import { NButton } from "naive-ui";
import { useI18n } from "vue-i18n";
import { cloneDeep } from "lodash-es";

import { ProjectMemberTable } from "../components/Project/ProjectSetting";
import {
  DEFAULT_PROJECT_ID,
  Principal,
  PrincipalId,
  Project,
  ProjectRoleType,
  UNKNOWN_ID,
} from "../types";
import { PrincipalSelect, ProjectRolesSelect } from "./v2";
import {
  addRoleToProjectIamPolicy,
  hasPermissionInProject,
  hasWorkspacePermission,
} from "../utils";
import {
  featureToRef,
  pushNotification,
  useCurrentUser,
  usePrincipalStore,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
} from "@/store";

interface LocalState {
  principalId: PrincipalId | undefined;
  roleList: ProjectRoleType[];
  adding: boolean;
}

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
});

const ROLE_DEVELOPER = "roles/DEVELOPER";
const { t } = useI18n();
const currentUser = useCurrentUser();
const projectResourceName = computed(
  () => `projects/${props.project.resourceId}`
);
const { policy: iamPolicy, ready } = useProjectIamPolicy(projectResourceName);

const state = reactive<LocalState>({
  principalId: undefined,
  roleList: [],
  adding: false,
});

const hasRBACFeature = featureToRef("bb.feature.rbac");

const allowAdmin = computed(() => {
  if (props.project.id === DEFAULT_PROJECT_ID) {
    return false;
  }

  if (props.project.rowStatus === "ARCHIVED") {
    return false;
  }

  // Allow workspace roles having manage project permission here in case project owners are not available.
  if (
    hasWorkspacePermission(
      "bb.permission.workspace.manage-project",
      currentUser.value.role
    )
  ) {
    return true;
  }

  if (
    hasPermissionInProject(
      props.project,
      currentUser.value,
      "bb.permission.project.manage-member"
    )
  ) {
    return true;
  }
  return false;
});

const isValid = computed(() => {
  const { principalId, roleList } = state;
  return principalId && principalId !== UNKNOWN_ID && roleList.length > 0;
});

const filterNonMemberUsers = (principal: Principal) => {
  const user = `user:${principal.email}`;
  return iamPolicy.value.bindings.every(
    (binding) => !binding.members.includes(user)
  );
};

const addMember = async () => {
  if (!isValid.value) return;
  const { principalId, roleList } = state;
  if (!principalId) return;
  state.adding = true;
  try {
    const principal = usePrincipalStore().principalById(principalId);
    const policy = cloneDeep(iamPolicy.value);
    const user = `user:${principal.email}`;
    roleList.forEach((role) => {
      addRoleToProjectIamPolicy(policy, user, role);
    });
    await useProjectIamPolicyStore().updateProjectIamPolicy(
      projectResourceName.value,
      policy
    );

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("project.settings.success-member-added-prompt", {
        name: principal.name,
      }),
    });
    state.roleList = [ROLE_DEVELOPER];
    state.principalId = undefined;
  } finally {
    state.adding = false;
  }
};
</script>
