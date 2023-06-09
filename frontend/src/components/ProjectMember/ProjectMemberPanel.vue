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
          v-model:users="state.userUIDList"
          :multiple="true"
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

import ProjectMemberTable, {
  ComposedProjectMember,
} from "./ProjectMemberTable";
import { DEFAULT_PROJECT_V1_NAME, PresetRoleType, unknownUser } from "@/types";
import { UserSelect } from "@/components/v2";
import {
  addRoleToProjectIamPolicy,
  extractUserUID,
  hasPermissionInProjectV1,
  hasWorkspacePermissionV1,
} from "@/utils";
import {
  extractUserEmail,
  featureToRef,
  pushNotification,
  useCurrentUserV1,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
  useUserStore,
} from "@/store";
import { Project } from "@/types/proto/v1/project_service";
import { State } from "@/types/proto/v1/common";
import { User } from "@/types/proto/v1/auth_service";

interface LocalState {
  userUIDList: string[];
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
const currentUserV1 = useCurrentUserV1();
const projectResourceName = computed(() => props.project.name);
const { policy: iamPolicy, ready } = useProjectIamPolicy(projectResourceName);

const state = reactive<LocalState>({
  userUIDList: [],
  roleList: [],
  adding: false,
  showInactiveMemberList: false,
});

const hasRBACFeature = featureToRef("bb.feature.rbac");
const userStore = useUserStore();

const allowAdmin = computed(() => {
  if (props.project.name === DEFAULT_PROJECT_V1_NAME) {
    return false;
  }

  if (props.project.state === State.DELETED) {
    return false;
  }

  // Allow workspace roles having manage project permission here in case project owners are not available.
  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-project",
      currentUserV1.value.userRole
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
  const { userUIDList, roleList } = state;
  return userUIDList.length > 0 && roleList.length > 0;
});

const filterNonMemberUsers = (user: User) => {
  return iamPolicy.value.bindings.every(
    (binding) => !binding.members.includes(`user:${user.email}`)
  );
};

const addMember = async () => {
  if (!isValid.value) return;
  const { userUIDList, roleList } = state;
  if (userUIDList.length === 0 || roleList.length === 0) return;
  state.adding = true;
  try {
    const policy = cloneDeep(iamPolicy.value);
    const userNameList = [];
    for (const userUID of userUIDList) {
      const user = userStore.getUserById(userUID) ?? unknownUser();
      userNameList.push(user.title);
      const tag = `user:${user.email}`;
      roleList.forEach((role) => {
        addRoleToProjectIamPolicy(policy, tag, role);
      });
    }
    await useProjectIamPolicyStore().updateProjectIamPolicy(
      projectResourceName.value,
      policy
    );

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("project.settings.success-member-added-prompt", {
        name: userNameList.join(", "),
      }),
    });
    state.roleList = [];
    state.userUIDList = [];
  } finally {
    state.adding = false;
  }
};
</script>
