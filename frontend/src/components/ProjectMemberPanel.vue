<template>
  <div class="">
    <FeatureAttention
      v-if="!hasRBACFeature"
      custom-class="my-5"
      feature="bb.feature.rbac"
      :description="$t('subscription.features.bb-feature-rbac.desc')"
    />
    <div class="text-lg font-medium leading-7 text-main">
      <span>{{ $t("project.settings.manage-member") }}</span>
      <span class="ml-1 font-normal text-control-light">
        ({{ activeComposedPrincipalList.length }})
      </span>
    </div>

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
      :editable="true"
      :composed-principal-list="activeComposedPrincipalList"
    />

    <div v-if="inactiveComposedPrincipalList.length > 0" class="mt-4">
      <NCheckbox v-model:checked="state.showInactiveMemberList">
        <span class="textinfolabel">
          {{ $t("project.settings.members.show-inactive") }}
        </span>
      </NCheckbox>
    </div>

    <div v-if="state.showInactiveMemberList" class="my-4 space-y-2">
      <div class="text-lg font-medium leading-7 text-main">
        <span>{{ $t("project.settings.members.inactive-members") }}</span>
        <span class="ml-1 font-normal text-control-light">
          ({{ inactiveComposedPrincipalList.length }})
        </span>
      </div>
      <ProjectMemberTable
        :iam-policy="iamPolicy"
        :project="project"
        :ready="ready"
        :editable="false"
        :composed-principal-list="
          composedPrincipalList.filter(
            (item) => item.member.rowStatus === 'ARCHIVED'
          )
        "
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive } from "vue";
import { NButton, NCheckbox } from "naive-ui";
import { useI18n } from "vue-i18n";
import { cloneDeep, orderBy, uniq } from "lodash-es";

import { ProjectMemberTable } from "../components/Project/ProjectSetting";
import {
  DEFAULT_PROJECT_ID,
  Principal,
  PrincipalId,
  ProjectRoleType,
  UNKNOWN_ID,
} from "../types";
import { PrincipalSelect, ProjectRolesSelect } from "./v2";
import {
  addRoleToProjectIamPolicy,
  hasPermissionInProjectV1,
  hasWorkspacePermission,
} from "../utils";
import {
  extractUserEmail,
  featureToRef,
  pushNotification,
  useCurrentUser,
  useCurrentUserV1,
  useMemberStore,
  usePrincipalStore,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
  useUserStore,
} from "@/store";
import { ComposedPrincipal } from "./Project/ProjectSetting/common";
import { Project } from "@/types/proto/v1/project_service";
import { State } from "@/types/proto/v1/common";

interface LocalState {
  principalId: PrincipalId | undefined;
  roleList: ProjectRoleType[];
  adding: boolean;
  showInactiveMemberList: boolean;
}

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
});

const ROLE_OWNER = "roles/OWNER";
const { t } = useI18n();
const currentUser = useCurrentUser();
const currentUserV1 = useCurrentUserV1();
const projectResourceName = computed(() => props.project.name);
const { policy: iamPolicy, ready } = useProjectIamPolicy(projectResourceName);

const state = reactive<LocalState>({
  principalId: undefined,
  roleList: [],
  adding: false,
  showInactiveMemberList: false,
});

const hasRBACFeature = featureToRef("bb.feature.rbac");
const userStore = useUserStore();
const memberStore = useMemberStore();

const allowAdmin = computed(() => {
  if (parseInt(props.project.uid, 10) === DEFAULT_PROJECT_ID) {
    return false;
  }

  if (props.project.state === State.DELETED) {
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
    hasPermissionInProjectV1(
      iamPolicy.value,
      currentUserV1.value,
      "bb.permission.project.manage-member"
    )
  ) {
    return true;
  }
  return false;
});

const composedPrincipalList = computed(() => {
  const distinctUserResourceNameList = uniq(
    iamPolicy.value.bindings.flatMap((binding) => binding.members)
  );

  const userEmailList = distinctUserResourceNameList.map((user) =>
    extractUserEmail(user)
  );

  const composedUserList = userEmailList.map((email) => {
    const user = userStore.getUserByEmail(email);
    const member = memberStore.memberByEmail(email);
    return { email, user, member };
  });

  const usersByRole = iamPolicy.value.bindings.map((binding) => {
    return {
      role: binding.role,
      users: new Set(binding.members),
    };
  });
  const composedPrincipalList = composedUserList.map<ComposedPrincipal>(
    ({ email, member }) => {
      const resourceName = `user:${email}`;
      const roleList = usersByRole
        .filter((binding) => binding.users.has(resourceName))
        .map((binding) => binding.role);
      return {
        email,
        member,
        principal: member.principal,
        roleList,
      };
    }
  );

  return orderBy(
    composedPrincipalList,
    [
      (item) => (item.roleList.includes(ROLE_OWNER) ? 0 : 1),
      (item) => item.principal.id,
    ],
    ["asc", "asc"]
  );
});

const activeComposedPrincipalList = computed(() => {
  return composedPrincipalList.value.filter(
    (item) => item.member.rowStatus === "NORMAL"
  );
});

const inactiveComposedPrincipalList = computed(() => {
  return composedPrincipalList.value.filter(
    (item) => item.member.rowStatus === "ARCHIVED"
  );
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
    state.roleList = [];
    state.principalId = undefined;
  } finally {
    state.adding = false;
  }
};
</script>
