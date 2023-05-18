<template>
  <NDrawer
    :show="true"
    width="auto"
    :auto-focus="false"
    @update:show="(show) => !show && $emit('close')"
  >
    <NDrawerContent
      :title="panelTitle"
      :closable="true"
      class="w-[44rem] max-w-[100vw] relative"
    >
      <div v-for="role in roleList" :key="role.role" class="mb-4">
        <template v-if="role.formatedConditionList.length > 0">
          <div
            class="w-full px-2 py-2 flex flex-row justify-start items-center"
          >
            <span class="textlabel">{{ displayRoleTitle(role.role) }}</span>
            <NTooltip :disabled="allowRemoveRole(role.role)">
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
                {{ $t("project.settings.members.cannot-remove-last-owner") }}
              </div>
            </NTooltip>
          </div>
          <BBGrid
            :column-list="COLUMNS"
            :row-clickable="false"
            :data-source="role.formatedConditionList"
            class="border"
          >
            <template #item="{ item }: FormatedConditionRow">
              <div class="bb-grid-cell">
                {{ extractDatabaseName(item.database) }}
              </div>
              <div class="bb-grid-cell">
                {{ extractExpiration(item.expiration) }}
              </div>
              <div class="bb-grid-cell">
                <RoleDescription :description="item.description || ''" />
              </div>
              <div class="bb-grid-cell w-12">
                <button
                  v-if="item.database"
                  class="cursor-pointer opacity-60 hover:opacity-100"
                  @click="handleDeleteCondition(item)"
                >
                  <heroicons-outline:trash class="w-4 h-4" />
                </button>
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
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { NButton, NDrawer, NDrawerContent, useDialog } from "naive-ui";

import {
  ComposedPrincipal,
  DatabaseId,
  PresetRoleType,
  UNKNOWN_ID,
} from "@/types";
import {
  useCurrentUser,
  useCurrentUserV1,
  useDatabaseStore,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
  useUserStore,
} from "@/store";
import { Binding, Project } from "@/types/proto/v1/project_service";
import { watch } from "vue";
import {
  displayRoleTitle,
  getDatabaseIdByName,
  getDatabaseNameById,
  hasPermissionInProjectV1,
  hasWorkspacePermission,
  parseConditionExpressionString,
  stringifyConditionExpression,
} from "@/utils";
import { BBGridColumn, BBGrid, BBGridRow } from "@/bbkit";
import { useI18n } from "vue-i18n";
import RoleDescription from "./RoleDescription.vue";
import { cloneDeep, isEqual, uniq } from "lodash-es";
import { State } from "@/types/proto/v1/common";
import { getUserEmailFromIdentifier } from "@/store/modules/v1/common";

interface FormatedCondition {
  database?: DatabaseId;
  table?: string;
  expiration?: Date;
  description?: string;
  rawRole: Binding;
}

export type FormatedConditionRow = BBGridRow<FormatedCondition>;

const props = defineProps<{
  project: Project;
  member: ComposedPrincipal;
}>();

const emits = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const dialog = useDialog();
const currentUser = useCurrentUser();
const currentUserV1 = useCurrentUserV1();
const userStore = useUserStore();
const databaseStore = useDatabaseStore();
const projectIamPolicyStore = useProjectIamPolicyStore();
const projectResourceName = computed(() => props.project.name);
const { policy: iamPolicy } = useProjectIamPolicy(projectResourceName);
const roleList = ref<
  {
    role: string;
    formatedConditionList: FormatedCondition[];
  }[]
>([]);

const panelTitle = computed(() => {
  return t("project.settings.members.edit", {
    member: `${props.member.principal.name}(${props.member.email})`,
  });
});

const COLUMNS = computed(() => {
  const columns: BBGridColumn[] = [
    {
      title: t("common.database"),
      width: "1fr",
    },
    {
      title: t("common.expiration"),
      width: "1fr",
    },
    {
      title: t("common.description"),
      width: "1fr",
    },
    {
      title: "",
      width: "3rem",
    },
  ];
  return columns;
});

const allowAdmin = computed(() => {
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

const handleDeleteRole = (role: string) => {
  const title = t("project.settings.members.revoke-role-from-user", {
    role: displayRoleTitle(role),
    user: props.member.principal.name,
  });
  dialog.create({
    title: title,
    content: t("common.cannot-undo-this-action"),
    positiveText: t("common.revoke"),
    negativeText: t("common.cancel"),
    closable: false,
    maskClosable: false,
    closeOnEsc: false,
    onNegativeClick: () => {
      // nothing to do
    },
    onPositiveClick: async () => {
      const user = `user:${props.member.email}`;
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

const handleDeleteCondition = async (condition: FormatedCondition) => {
  if (!condition.database) {
    return;
  }

  const database = await databaseStore.getOrFetchDatabaseById(
    condition.database
  );
  const title = t("project.settings.members.revoke-role-from-user", {
    role: `${displayRoleTitle(condition.rawRole.role)} - ${database.name}`,
    user: props.member.principal.name,
  });
  dialog.create({
    title: title,
    content: t("common.cannot-undo-this-action"),
    positiveText: t("common.revoke"),
    negativeText: t("common.cancel"),
    closable: false,
    maskClosable: false,
    closeOnEsc: false,
    onNegativeClick: () => {
      // nothing to do
    },
    onPositiveClick: async () => {
      const user = `user:${props.member.email}`;
      const policy = cloneDeep(iamPolicy.value);
      let rawRole = policy.bindings.find((binding) =>
        isEqual(binding, condition.rawRole)
      );
      if (!rawRole || !rawRole.condition) {
        return;
      }
      if (rawRole.members.length > 1) {
        rawRole.members = rawRole.members.filter((member) => {
          return member !== user;
        });
        rawRole = cloneDeep(rawRole);
        rawRole.members = [user];
        policy.bindings.push(rawRole);
      }
      // Update rawRole's expression.
      const conditionExpression = parseConditionExpressionString(
        rawRole.condition?.expression || ""
      );
      if (conditionExpression.databases === undefined) {
        return;
      }
      const databaseName = await getDatabaseNameById(condition.database!);
      conditionExpression.databases = conditionExpression.databases.filter(
        (name) => name !== databaseName
      );
      if (conditionExpression.databases.length === 0) {
        policy.bindings = policy.bindings.filter(
          (binding) => !isEqual(binding, rawRole)
        );
      } else {
        rawRole.condition!.expression =
          stringifyConditionExpression(conditionExpression);
      }
      await projectIamPolicyStore.updateProjectIamPolicy(
        projectResourceName.value,
        policy
      );
    },
  });
};

const extractDatabaseName = (databaseId?: DatabaseId) => {
  if (!databaseId) {
    return "*";
  }
  const database = databaseStore.getDatabaseById(databaseId);
  return database.name;
};

const extractExpiration = (expiration?: Date) => {
  if (!expiration) {
    return "*";
  }
  return expiration.toLocaleString();
};

const parseExpression = async (rawExpression: string) => {
  const condition: {
    database?: DatabaseId[];
    expiredAt?: Date;
  } = {
    database: undefined,
    expiredAt: undefined,
  };

  const conditionExpression = parseConditionExpressionString(rawExpression);
  if (conditionExpression.databases !== undefined) {
    const databaseIdList = [];
    for (const name of conditionExpression.databases) {
      const databaseId = await getDatabaseIdByName(name);
      databaseIdList.push(databaseId);
    }
    const filteredDatabaseIdList = uniq(
      databaseIdList.filter((databaseId) => databaseId !== UNKNOWN_ID)
    );
    if (filteredDatabaseIdList.length > 0) {
      condition.database = filteredDatabaseIdList;
    }
  }
  if (conditionExpression.expiredTime !== undefined) {
    condition.expiredAt = new Date(conditionExpression.expiredTime);
  }

  return condition;
};

watch(
  () => [iamPolicy.value?.bindings],
  async () => {
    const tempRoleList: {
      role: string;
      formatedConditionList: FormatedCondition[];
    }[] = [];
    const rawRoleList = iamPolicy.value?.bindings?.filter((binding) => {
      return binding.members.includes(`user:${props.member.email}`);
    });
    for (const rawRole of rawRoleList) {
      const formatedConditionList = [];
      const parsedCondition = await parseExpression(
        rawRole.condition?.expression || ""
      );
      const description = rawRole.condition?.description || "";
      if (Array.isArray(parsedCondition.database)) {
        for (const database of parsedCondition.database) {
          formatedConditionList.push({
            database: database,
            table: undefined,
            expiration: parsedCondition.expiredAt,
            description: description,
            rawRole: rawRole,
          });
        }
      } else {
        formatedConditionList.push({
          database: undefined,
          table: undefined,
          expiration: undefined,
          description: description,
          rawRole: rawRole,
        });
      }
      const tempRole = tempRoleList.find((role) => role.role === rawRole.role);
      if (tempRole) {
        tempRole.formatedConditionList.push(...formatedConditionList);
      } else {
        tempRoleList.push({
          role: rawRole.role,
          formatedConditionList,
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
