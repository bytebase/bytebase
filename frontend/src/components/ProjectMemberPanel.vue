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
        ({{ activeComposedMemberList.length }})
      </span>
    </div>

    <div v-if="allowAdmin" class="my-4 w-full flex gap-x-2">
      <div class="w-[18rem] shrink-0">
        <UserSelect
          v-model:user="state.userUID"
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
      :member-list="activeComposedMemberList"
    />

    <div v-if="inactiveComposedMemberList.length > 0" class="mt-4">
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
          ({{ inactiveComposedMemberList.length }})
        </span>
      </div>
      <ProjectMemberTable
        :iam-policy="iamPolicy"
        :project="project"
        :ready="ready"
        :editable="false"
        :member-list="inactiveComposedMemberList"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive } from "vue";
import { NButton, NCheckbox } from "naive-ui";
import { useI18n } from "vue-i18n";
import { cloneDeep, orderBy, uniq } from "lodash-es";

import {
  ComposedProjectMember,
  ProjectMemberTable,
} from "../components/Project/ProjectSetting";
import { DEFAULT_PROJECT_ID, PresetRoleType, unknownUser } from "../types";
import { UserSelect } from "./v2";
import {
  addRoleToProjectIamPolicy,
  extractUserUID,
  hasPermissionInProjectV1,
  hasWorkspacePermission,
} from "../utils";
import {
  extractUserEmail,
  featureToRef,
  pushNotification,
  useCurrentUser,
  useCurrentUserV1,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
  useUserStore,
} from "@/store";
import { Project } from "@/types/proto/v1/project_service";
import { State } from "@/types/proto/v1/common";
import { User } from "@/types/proto/v1/auth_service";

interface LocalState {
  userUID: string | undefined;
  roleList: string[];
  adding: boolean;
  showInactiveMemberList: boolean;
}

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
});

const { t } = useI18n();
const currentUser = useCurrentUser();
const currentUserV1 = useCurrentUserV1();
const projectResourceName = computed(() => props.project.name);
const { policy: iamPolicy, ready } = useProjectIamPolicy(projectResourceName);

const state = reactive<LocalState>({
  userUID: undefined,
  roleList: [],
  adding: false,
  showInactiveMemberList: false,
});

const hasRBACFeature = featureToRef("bb.feature.rbac");
const userStore = useUserStore();

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

const composedMemberList = computed(() => {
  const distinctUserResourceNameList = uniq(
    iamPolicy.value.bindings.flatMap((binding) => binding.members)
  );

  const userList = distinctUserResourceNameList.map((user) => {
    const email = extractUserEmail(user);
    return (
      userStore.getUserByEmail(email) ?? {
        ...unknownUser(),
        email,
      }
    );
  });

  const usersByRole = iamPolicy.value.bindings.map((binding) => {
    return {
      role: binding.role,
      users: new Set(binding.members.map(extractUserEmail)),
    };
  });
  const userRolesList = userList.map<ComposedProjectMember>((user) => {
    const roleList = uniq(
      usersByRole
        .filter((binding) => binding.users.has(user.email))
        .map((binding) => binding.role)
    );
    return {
      user,
      roleList,
    };
  });

  return orderBy(
    userRolesList,
    [
      (item) => (item.roleList.includes(PresetRoleType.OWNER) ? 0 : 1),
      (item) => parseInt(extractUserUID(item.user.name), 10),
    ],
    ["asc", "asc"]
  );
});

const activeComposedMemberList = computed(() => {
  return composedMemberList.value.filter(
    (item) => item.user.state === State.ACTIVE
  );
});

const inactiveComposedMemberList = computed(() => {
  return composedMemberList.value.filter(
    (item) => item.user.state === State.DELETED
  );
});

const isValid = computed(() => {
  const { userUID, roleList } = state;
  return userUID && roleList.length > 0;
});

const filterNonMemberUsers = (user: User) => {
  return iamPolicy.value.bindings.every(
    (binding) => !binding.members.includes(`user:${user.email}`)
  );
};

const addMember = async () => {
  if (!isValid.value) return;
  const { userUID, roleList } = state;
  if (!userUID) return;
  state.adding = true;
  try {
    const policy = cloneDeep(iamPolicy.value);
    const user = userStore.getUserById(userUID) ?? unknownUser();
    const tag = `user:${user.email}`;
    roleList.forEach((role) => {
      addRoleToProjectIamPolicy(policy, tag, role);
    });
    await useProjectIamPolicyStore().updateProjectIamPolicy(
      projectResourceName.value,
      policy
    );

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("project.settings.success-member-added-prompt", {
        name: user.title,
      }),
    });
    state.roleList = [];
    state.userUID = undefined;
  } finally {
    state.adding = false;
  }
};
</script>
