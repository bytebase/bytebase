<template>
  <NDrawer
    :show="true"
    width="auto"
    :auto-focus="false"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <NDrawerContent
      :title="panelTitle"
      :closable="true"
      class="w-[64rem] max-w-[100vw] relative"
    >
      <div v-for="role in roleList" :key="role.role" class="mb-4">
        <template v-if="role.singleBindingList.length > 0">
          <div
            class="w-full px-2 py-2 flex flex-row justify-start items-center"
          >
            <span class="textlabel">{{ displayRoleTitle(role.role) }}</span>
            <NTooltip
              v-if="allowAdmin"
              :disabled="
                allowRemoveRole(role.role) || role.role !== 'roles/OWNER'
              "
            >
              <template #trigger>
                <NButton
                  tag="div"
                  text
                  class="cursor-pointer opacity-60 hover:opacity-100"
                  :disabled="!allowRemoveRole(role.role)"
                  @click="handleDeleteRole(role.role)"
                >
                  <heroicons-outline:trash class="w-4 h-4 ml-1" />
                </NButton>
              </template>
              <div>
                {{ $t("project.members.cannot-remove-last-owner") }}
              </div>
            </NTooltip>
          </div>
          <BBGrid
            :column-list="COLUMNS"
            :row-clickable="false"
            :data-source="role.singleBindingList"
            class="border"
          >
            <template #item="{ item }: SingleBindingRow">
              <div class="bb-grid-cell !p-0 items-center justify-center">
                <RoleExpiredTip v-if="checkRoleExpired(item)" />
              </div>
              <div class="bb-grid-cell">
                <span class="shrink-0 mr-1">{{
                  extractDatabaseName(item.databaseResource)
                }}</span>
                <template v-if="item.databaseResource">
                  <InstanceV1Name
                    class="text-gray-500"
                    :instance="
                      extractDatabase(item.databaseResource).instanceEntity
                    "
                    :link="false"
                  />
                </template>
              </div>
              <div class="bb-grid-cell">
                {{ extractSchemaName(item.databaseResource) }}
              </div>
              <div class="bb-grid-cell">
                {{ extractTableName(item.databaseResource) }}
              </div>
              <div class="bb-grid-cell">
                {{ extractExpiration(item.expiration) }}
              </div>
              <div class="bb-grid-cell">
                <RoleDescription :description="item.description || ''" />
              </div>
              <div class="bb-grid-cell space-x-1">
                <NTooltip v-if="allowAdmin" trigger="hover">
                  <template #trigger>
                    <button
                      class="cursor-pointer opacity-60 hover:opacity-100"
                      @click="editingBinding = item"
                    >
                      <heroicons-outline:pencil class="w-4 h-4" />
                    </button>
                  </template>
                  {{ $t("common.edit") }}
                </NTooltip>
                <NTooltip v-if="allowAdmin" trigger="hover">
                  <template #trigger>
                    <button
                      class="cursor-pointer opacity-60 hover:opacity-100"
                      @click="handleDeleteCondition(item)"
                    >
                      <heroicons-outline:trash class="w-4 h-4" />
                    </button>
                  </template>
                  {{ $t("common.delete") }}
                </NTooltip>
              </div>
            </template>
          </BBGrid>
        </template>
      </div>
      <template #footer>
        <div class="flex items-center justify-end gap-x-2">
          <NButton @click="$emit('close')">{{ $t("common.cancel") }}</NButton>
          <NButton type="primary" @click="$emit('close')">
            {{ $t("common.ok") }}
          </NButton>
        </div>
      </template>
    </NDrawerContent>
  </NDrawer>

  <EditProjectMemberPanel
    v-if="editingBinding"
    :project="project"
    :member="member"
    :single-binding="editingBinding"
    @close="editingBinding = null"
  />
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual } from "lodash-es";
import {
  NButton,
  NDrawer,
  NDrawerContent,
  NTooltip,
  useDialog,
} from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

import { ComposedProject, DatabaseResource, PresetRoleType } from "@/types";
import { State } from "@/types/proto/v1/common";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
  useUserStore,
} from "@/store";
import {
  displayRoleTitle,
  hasPermissionInProjectV1,
  hasWorkspacePermissionV1,
} from "@/utils";
import {
  convertFromExpr,
  stringifyConditionExpression,
} from "@/utils/issue/cel";
import { ComposedProjectMember, SingleBinding } from "./types";
import { BBGridColumn, BBGrid, BBGridRow } from "@/bbkit";
import RoleDescription from "./RoleDescription.vue";
import EditProjectMemberPanel from "../AddProjectMember/EditProjectMemberPanel.vue";
import RoleExpiredTip from "./RoleExpiredTip.vue";

export type SingleBindingRow = BBGridRow<SingleBinding>;

const props = defineProps<{
  project: ComposedProject;
  member: ComposedProjectMember;
}>();

const emits = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const dialog = useDialog();
const currentUserV1 = useCurrentUserV1();
const userStore = useUserStore();
const databaseStore = useDatabaseV1Store();
const projectIamPolicyStore = useProjectIamPolicyStore();
const projectResourceName = computed(() => props.project.name);
const { policy: iamPolicy } = useProjectIamPolicy(projectResourceName);
const roleList = ref<
  {
    role: string;
    singleBindingList: SingleBinding[];
  }[]
>([]);
const editingBinding = ref<SingleBinding | null>(null);

const panelTitle = computed(() => {
  return t("project.members.edit", {
    member: `${props.member.user.title}(${props.member.user.email})`,
  });
});

const COLUMNS = computed(() => {
  const columns: BBGridColumn[] = [
    {
      title: "",
      width: "2rem",
    },
    {
      title: t("common.database"),
      width: "2fr",
    },
    {
      title: t("common.schema"),
      width: "6rem",
    },
    {
      title: t("common.table"),
      width: "6rem",
    },
    {
      title: t("common.expiration"),
      width: "12rem",
    },
    {
      title: t("common.description"),
      width: "6rem",
    },
    {
      title: "",
      width: "4rem",
    },
  ];
  return columns;
});

const allowAdmin = computed(() => {
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

// To prevent user accidentally removing roles and lock the project permanently, we take following measures:
// 1. Disallow removing the last OWNER.
// 2. Allow workspace roles who can manage project. This helps when the project OWNER is no longer available.
const allowRemoveRole = (role: string) => {
  if (props.project.state === State.DELETED) {
    return false;
  }

  if (role === PresetRoleType.OWNER) {
    const binding = iamPolicy.value.bindings.find(
      (binding) => binding.role === PresetRoleType.OWNER
    );
    const members = (binding?.members || [])
      .map((userIdentifier) => {
        return userStore.getUserByIdentifier(userIdentifier);
      })
      .filter((user) => user?.state === State.ACTIVE);
    if (!binding || members.length === 1) {
      return false;
    }
  }

  return allowAdmin.value;
};

const handleDeleteRole = (role: string) => {
  const title = t("project.members.revoke-role-from-user", {
    role: displayRoleTitle(role),
    user: props.member.user.title,
  });
  dialog.create({
    title: title,
    content: t("common.cannot-undo-this-action"),
    positiveText: t("common.revoke"),
    negativeText: t("common.cancel"),
    onPositiveClick: async () => {
      const user = `user:${props.member.user.email}`;
      const policy = cloneDeep(iamPolicy.value);
      for (const binding of policy.bindings) {
        if (binding.role !== role) {
          continue;
        }
        if (binding.members.includes(user)) {
          binding.members = binding.members.filter((member) => {
            return member !== user;
          });
        }
        if (binding.members.length === 0) {
          policy.bindings = policy.bindings.filter(
            (item) => !isEqual(item, binding)
          );
        }
      }
      await projectIamPolicyStore.updateProjectIamPolicy(
        projectResourceName.value,
        policy
      );
    },
  });
};

const handleDeleteCondition = async (singleBinding: SingleBinding) => {
  let role = `${displayRoleTitle(singleBinding.rawBinding.role)}`;
  if (singleBinding.databaseResource) {
    const database = await databaseStore.getOrFetchDatabaseByName(
      String(singleBinding.databaseResource.databaseName)
    );
    role = `${role} - ${database.databaseName}`;
  }
  const title = t("project.members.revoke-role-from-user", {
    role: role,
    user: props.member.user.title,
  });
  dialog.create({
    title: title,
    content: t("common.cannot-undo-this-action"),
    positiveText: t("common.revoke"),
    negativeText: t("common.cancel"),
    onPositiveClick: async () => {
      const user = `user:${props.member.user.email}`;
      const policy = cloneDeep(iamPolicy.value);
      const rawBinding = policy.bindings.find((binding) =>
        isEqual(binding, singleBinding.rawBinding)
      );
      if (!rawBinding) {
        return;
      }

      rawBinding.members = rawBinding.members.filter((member) => {
        return member !== user;
      });

      if (rawBinding.parsedExpr?.expr) {
        const conditionExpr = convertFromExpr(rawBinding.parsedExpr.expr);
        if (conditionExpr.databaseResources) {
          conditionExpr.databaseResources =
            conditionExpr.databaseResources.filter(
              (resource) => !isEqual(resource, singleBinding.databaseResource)
            );
          if (conditionExpr.databaseResources.length === 0) {
            policy.bindings = policy.bindings.filter(
              (binding) => !isEqual(binding, rawBinding)
            );
          } else {
            const newBinding = cloneDeep(rawBinding);
            newBinding.members = [user];
            newBinding.condition!.expression =
              stringifyConditionExpression(conditionExpr);
            policy.bindings.push(newBinding);
          }
        }
      }

      if (rawBinding.members.length === 0) {
        policy.bindings = policy.bindings.filter(
          (binding) => !isEqual(binding, rawBinding)
        );
      }

      await projectIamPolicyStore.updateProjectIamPolicy(
        projectResourceName.value,
        policy
      );
    },
  });
};

const extractDatabaseName = (databaseResource?: DatabaseResource) => {
  if (!databaseResource) {
    return "*";
  }
  const database = databaseStore.getDatabaseByName(
    String(databaseResource.databaseName)
  );
  return database.databaseName;
};

const extractDatabase = (databaseResource: DatabaseResource) => {
  const database = databaseStore.getDatabaseByName(
    String(databaseResource.databaseName)
  );
  return database;
};

const extractSchemaName = (databaseResource?: DatabaseResource) => {
  if (!databaseResource) {
    return "*";
  }

  if (databaseResource.schema === undefined) {
    return "*";
  } else if (databaseResource.schema === "") {
    return "-";
  } else {
    return databaseResource.schema;
  }
};

const extractTableName = (databaseResource?: DatabaseResource) => {
  if (!databaseResource) {
    return "*";
  }

  if (databaseResource.table === undefined) {
    return "*";
  } else if (databaseResource.table === "") {
    return "-";
  } else {
    return databaseResource.table;
  }
};

const extractExpiration = (expiration?: Date) => {
  if (!expiration) {
    return "*";
  }
  return expiration.toLocaleString();
};

const checkRoleExpired = (role: SingleBinding) => {
  if (!role.expiration) {
    return false;
  }
  return role.expiration < new Date();
};

watch(
  () => [iamPolicy.value?.bindings],
  async () => {
    const tempRoleList: {
      role: string;
      singleBindingList: SingleBinding[];
    }[] = [];
    const rawBindingList = iamPolicy.value?.bindings?.filter((binding) => {
      return binding.members.includes(`user:${props.member.user.email}`);
    });
    for (const rawBinding of rawBindingList) {
      const singleBindingList = [];
      const singleBinding: SingleBinding = {
        databaseResource: undefined,
        expiration: undefined,
        description: undefined,
        rawBinding: rawBinding,
      };

      if (rawBinding.parsedExpr?.expr) {
        const conditionExpr = convertFromExpr(rawBinding.parsedExpr.expr);
        singleBinding.description = rawBinding.condition?.description || "";
        if (conditionExpr.expiredTime) {
          singleBinding.expiration = new Date(conditionExpr.expiredTime);
        }
        if (
          Array.isArray(conditionExpr.databaseResources) &&
          conditionExpr.databaseResources.length > 0
        ) {
          for (const resource of conditionExpr.databaseResources) {
            singleBindingList.push({
              ...singleBinding,
              databaseResource: resource,
            });
          }
        } else {
          singleBindingList.push(singleBinding);
        }
      } else {
        singleBindingList.push(singleBinding);
      }

      const tempRole = tempRoleList.find(
        (role) => role.role === rawBinding.role
      );
      if (tempRole) {
        tempRole.singleBindingList.push(...singleBindingList);
      } else {
        tempRoleList.push({
          role: rawBinding.role,
          singleBindingList: singleBindingList,
        });
      }
    }

    if (tempRoleList.length === 0) {
      emits("close");
    }
    roleList.value = tempRoleList;
  },
  {
    immediate: true,
  }
);
</script>
