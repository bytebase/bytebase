<template>
  <BBGrid
    :column-list="columnList"
    :row-clickable="false"
    :data-source="composedPrincipalList"
    class="border"
    row-key="email"
  >
    <template #item="{ item }: ComposedPrincipalRow">
      <div class="bb-grid-cell gap-x-2">
        <PrincipalAvatar :principal="item.principal" />
        <div class="flex flex-col">
          <div class="flex flex-row items-center space-x-2">
            <router-link :to="`/u/${item.principal.id}`" class="normal-link">{{
              item.principal.name
            }}</router-link>
            <span
              v-if="currentUser.id == item.principal.id"
              class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
              >{{ $t("common.you") }}</span
            >
          </div>
          <span class="textlabel">{{ item.principal.email }}</span>
        </div>
      </div>
      <div v-if="hasRBACFeature" class="bb-grid-cell flex-wrap gap-x-2 gap-y-1">
        <NTag
          v-for="role in item.roleList"
          :key="role"
          :closable="allowRemoveRole(role)"
          @close="removeRole(item, role)"
        >
          {{ displayRoleTitle(role) }}
        </NTag>
        <NPopselect
          v-if="allowAddRole(item)"
          :options="getRoleOptions(item)"
          :scrollable="true"
          trigger="click"
          @update:value="(value) => addRole(item, value)"
        >
          <NButton quaternary size="tiny">
            <heroicons-outline:plus />
          </NButton>
        </NPopselect>
      </div>
      <div class="bb-grid-cell">
        <NTooltip v-if="allowAdmin" :disabled="allowRemovePrincipal(item)">
          <template #trigger>
            <NButton
              tag="div"
              text
              :disabled="!allowRemovePrincipal(item)"
              @click="removePrincipal(item)"
            >
              <heroicons-outline:trash class="w-4 h-4" />
            </NButton>
          </template>

          <div>
            {{ $t("project.settings.members.cannot-remove-last-owner") }}
          </div>
        </NTooltip>
      </div>
    </template>

    <template #placeholder-content>
      <div class="p-2">-</div>
    </template>
  </BBGrid>
</template>

<script setup lang="ts">
import { computed } from "vue";
import {
  NButton,
  NPopselect,
  NTag,
  NTooltip,
  SelectOption,
  useDialog,
} from "naive-ui";
import { cloneDeep } from "lodash-es";
import { useI18n } from "vue-i18n";

import { ComposedPrincipal, ProjectRoleType } from "@/types";
import { type BBGridColumn, type BBGridRow, BBGrid } from "@/bbkit";
import { IamPolicy, Project } from "@/types/proto/v1/project_service";
import {
  featureToRef,
  useCurrentUser,
  useCurrentUserV1,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
  useRoleStore,
  useUserStore,
} from "@/store";
import {
  hasWorkspacePermission,
  displayRoleTitle,
  addRoleToProjectIamPolicy,
  removeRoleFromProjectIamPolicy,
  removeUserFromProjectIamPolicy,
  hasPermissionInProjectV1,
} from "@/utils";
import { State } from "@/types/proto/v1/common";
import { getUserEmailFromIdentifier } from "@/store/modules/v1/common";

export type ComposedPrincipalRow = BBGridRow<ComposedPrincipal>;

const props = defineProps<{
  project: Project;
  iamPolicy: IamPolicy;
  editable: boolean;
  composedPrincipalList: ComposedPrincipal[];
}>();

const ROLE_OWNER = "roles/OWNER";
const { t } = useI18n();
const hasRBACFeature = featureToRef("bb.feature.rbac");
const currentUser = useCurrentUser();
const currentUserV1 = useCurrentUserV1();
const userStore = useUserStore();
const roleStore = useRoleStore();
const projectIamPolicyStore = useProjectIamPolicyStore();
const dialog = useDialog();

const projectResourceName = computed(() => props.project.name);
const { policy: iamPolicy } = useProjectIamPolicy(projectResourceName);

const columnList = computed(() => {
  const ACCOUNT: BBGridColumn = {
    title: t("settings.members.table.account"),
    width: hasRBACFeature.value ? "minmax(auto, 18rem)" : "1fr",
  };
  const ROLE: BBGridColumn = {
    title: t("settings.members.table.roles"),
    width: "1fr",
  };
  const OPERATIONS: BBGridColumn = {
    title: "",
    width: "6rem",
  };
  if (hasRBACFeature.value) {
    return [ACCOUNT, ROLE, OPERATIONS];
  }
  return [ACCOUNT, OPERATIONS];
});

const allowAdmin = computed(() => {
  if (!props.editable) {
    return false;
  }

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

// To prevent user accidentally removing roles and lock the project permanently, we take following measures:
// 1. Disallow removing the last OWNER.
// 2. Allow workspace roles who can manage project. This helps when the project OWNER is no longer available.
const allowRemoveRole = (role: ProjectRoleType) => {
  if (props.project.state === State.DELETED) {
    return false;
  }

  if (role === ROLE_OWNER) {
    const binding = props.iamPolicy.bindings.find(
      (binding) => binding.role === ROLE_OWNER
    );
    const members = (binding?.members || [])
      .map((userIdentifier) => {
        const email = getUserEmailFromIdentifier(userIdentifier);
        return userStore.getUserByEmail(email);
      })
      .filter((user) => user?.state === State.ACTIVE);
    if (!binding || members.length === 1) {
      return false;
    }
  }

  return allowAdmin.value;
};

const removeRole = (item: ComposedPrincipal, role: string) => {
  const title = t("project.settings.members.revoke-role-from-user", {
    role: displayRoleTitle(role),
    user: item.principal.name,
  });
  const d = dialog.error({
    title,
    content: t("common.cannot-undo-this-action"),
    positiveText: t("common.revoke"),
    negativeText: t("common.cancel"),
    autoFocus: false,
    onPositiveClick: async () => {
      d.loading = true;
      const user = `user:${item.email}`;
      const policy = cloneDeep(props.iamPolicy);
      removeRoleFromProjectIamPolicy(policy, user, role);
      await projectIamPolicyStore.updateProjectIamPolicy(
        projectResourceName.value,
        policy
      );
    },
  });
};

const allowAddRole = (item: ComposedPrincipal) => {
  if (!allowAdmin.value) return false;
  if (props.project.state === State.DELETED) {
    return false;
  }

  return item.roleList.length < roleStore.roleList.length;
};

const getRoleOptions = (item: ComposedPrincipal) => {
  // TODO(steven): We don't allow to add EXPORTER and QUERIER roles directly for now.
  const roleList = useRoleStore().roleList.filter((role) => {
    return (
      role.name !== "roles/EXPORTER" &&
      role.name !== "roles/QUERIER" &&
      !item.roleList.includes(role.name)
    );
  });
  return roleList.map<SelectOption>((role) => {
    return {
      label: displayRoleTitle(role.name),
      value: role.name,
    };
  });
};

const addRole = async (item: ComposedPrincipal, role: string) => {
  const user = `user:${item.email}`;
  const policy = cloneDeep(props.iamPolicy);
  addRoleToProjectIamPolicy(policy, user, role);
  await projectIamPolicyStore.updateProjectIamPolicy(
    projectResourceName.value,
    policy
  );
};

const allowRemovePrincipal = (item: ComposedPrincipal) => {
  if (props.project.state === State.DELETED) {
    return false;
  }

  if (item.roleList.includes(ROLE_OWNER)) {
    const binding = props.iamPolicy.bindings.find(
      (binding) => binding.role === ROLE_OWNER
    );
    const members = (binding?.members || [])
      .map((userIdentifier) => {
        const email = getUserEmailFromIdentifier(userIdentifier);
        return userStore.getUserByEmail(email);
      })
      .filter((user) => user?.state === State.ACTIVE);
    if (!binding || members.length === 1) {
      return false;
    }
  }

  return allowAdmin.value;
};

const removePrincipal = (item: ComposedPrincipal) => {
  const d = dialog.error({
    title: t("project.settings.members.remove-user", {
      user: item.principal.name,
    }),
    content: t("common.cannot-undo-this-action"),
    positiveText: "Remove",
    negativeText: "Cancel",
    autoFocus: false,
    onPositiveClick: async () => {
      d.loading = true;
      const user = `user:${item.email}`;
      const policy = cloneDeep(props.iamPolicy);
      removeUserFromProjectIamPolicy(policy, user);
      await projectIamPolicyStore.updateProjectIamPolicy(
        projectResourceName.value,
        policy
      );
    },
  });
};
</script>
