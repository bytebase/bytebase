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
      class="w-[60rem] max-w-[100vw] relative"
    >
      <div v-for="role in roleList" :key="role.role" class="mb-4">
        <template v-if="role.formattedConditionList.length > 0">
          <div
            class="w-full px-2 py-2 flex flex-row justify-start items-center"
          >
            <span class="textlabel">{{ displayRoleTitle(role.role) }}</span>
            <NTooltip
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
                {{ $t("project.settings.members.cannot-remove-last-owner") }}
              </div>
            </NTooltip>
          </div>
          <BBGrid
            :column-list="COLUMNS"
            :row-clickable="false"
            :data-source="role.formattedConditionList"
            class="border"
          >
            <template #item="{ item }: FormattedConditionRow">
              <div class="bb-grid-cell">
                {{ extractDatabaseName(item.databaseResource) }}
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
              <div class="bb-grid-cell w-12">
                <button
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

import { PresetRoleType } from "@/types";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
  useUserStore,
} from "@/store";
import { Binding, Project } from "@/types/proto/v1/project_service";
import { watch } from "vue";
import {
  displayRoleTitle,
  hasPermissionInProjectV1,
  hasWorkspacePermissionV1,
} from "@/utils";
import { BBGridColumn, BBGrid, BBGridRow } from "@/bbkit";
import { useI18n } from "vue-i18n";
import RoleDescription from "./RoleDescription.vue";
import { cloneDeep, isEqual } from "lodash-es";
import { State } from "@/types/proto/v1/common";
import { ComposedProjectMember } from "./types";
import {
  convertFromExpr,
  stringifyConditionExpression,
} from "@/utils/issue/cel";
import { DatabaseResource } from "@/components/Issue/form/SelectDatabaseResourceForm/common";

export interface FormattedCondition {
  databaseResource?: DatabaseResource;
  expiration?: Date;
  description?: string;
  rawRole: Binding;
}

export type FormattedConditionRow = BBGridRow<FormattedCondition>;

const props = defineProps<{
  project: Project;
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
    formattedConditionList: FormattedCondition[];
  }[]
>([]);

const panelTitle = computed(() => {
  return t("project.settings.members.edit", {
    member: `${props.member.user.title}(${props.member.user.email})`,
  });
});

const COLUMNS = computed(() => {
  const columns: BBGridColumn[] = [
    {
      title: t("common.database"),
      width: "1fr",
    },
    {
      title: t("common.schema"),
      width: "1fr",
    },
    {
      title: t("common.table"),
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
  const title = t("project.settings.members.revoke-role-from-user", {
    role: displayRoleTitle(role),
    user: props.member.user.title,
  });
  dialog.create({
    title: title,
    content: t("common.cannot-undo-this-action"),
    positiveText: t("common.revoke"),
    negativeText: t("common.cancel"),
    autoFocus: false,
    closable: false,
    maskClosable: false,
    closeOnEsc: false,
    onNegativeClick: () => {
      // nothing to do
    },
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

const handleDeleteCondition = async (condition: FormattedCondition) => {
  let role = `${displayRoleTitle(condition.rawRole.role)}`;
  if (condition.databaseResource) {
    const database = await databaseStore.getOrFetchDatabaseByName(
      String(condition.databaseResource.databaseName)
    );
    role = `${role} - ${database.name}`;
  }
  const title = t("project.settings.members.revoke-role-from-user", {
    role: role,
    user: props.member.user.title,
  });
  dialog.create({
    title: title,
    content: t("common.cannot-undo-this-action"),
    positiveText: t("common.revoke"),
    negativeText: t("common.cancel"),
    autoFocus: false,
    closable: false,
    maskClosable: false,
    closeOnEsc: false,
    onNegativeClick: () => {
      // nothing to do
    },
    onPositiveClick: async () => {
      const user = `user:${props.member.user.email}`;
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

      if (rawRole.parsedExpr?.expr) {
        const conditionExpr = convertFromExpr(rawRole.parsedExpr.expr);
        if (conditionExpr.databaseResources) {
          conditionExpr.databaseResources =
            conditionExpr.databaseResources.filter(
              (resource) => !isEqual(resource, condition.databaseResource)
            );
          if (conditionExpr.databaseResources.length === 0) {
            policy.bindings = policy.bindings.filter(
              (binding) => !isEqual(binding, rawRole)
            );
          } else {
            rawRole.condition!.expression =
              stringifyConditionExpression(conditionExpr);
          }
        }
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

watch(
  () => [iamPolicy.value?.bindings],
  async () => {
    const tempRoleList: {
      role: string;
      formattedConditionList: FormattedCondition[];
    }[] = [];
    const rawRoleList = iamPolicy.value?.bindings?.filter((binding) => {
      return binding.members.includes(`user:${props.member.user.email}`);
    });
    for (const rawRole of rawRoleList) {
      const formattedConditionList = [];
      const formatedCondition: FormattedCondition = {
        databaseResource: undefined,
        expiration: undefined,
        description: undefined,
        rawRole: rawRole,
      };

      if (rawRole.parsedExpr?.expr) {
        const conditionExpr = convertFromExpr(rawRole.parsedExpr.expr);
        formatedCondition.description = rawRole.condition?.description || "";
        if (conditionExpr.expiredTime) {
          formatedCondition.expiration = new Date(conditionExpr.expiredTime);
        }
        if (
          Array.isArray(conditionExpr.databaseResources) &&
          conditionExpr.databaseResources.length > 0
        ) {
          for (const resource of conditionExpr.databaseResources) {
            formattedConditionList.push({
              ...formatedCondition,
              databaseResource: resource,
            });
          }
        } else {
          formattedConditionList.push(formatedCondition);
        }
      } else {
        formattedConditionList.push(formatedCondition);
      }

      const tempRole = tempRoleList.find((role) => role.role === rawRole.role);
      if (tempRole) {
        tempRole.formattedConditionList.push(...formattedConditionList);
      } else {
        tempRoleList.push({
          role: rawRole.role,
          formattedConditionList: formattedConditionList,
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
