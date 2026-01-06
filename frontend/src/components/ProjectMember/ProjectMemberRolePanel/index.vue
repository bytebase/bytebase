<template>
  <Drawer
    :show="true"
    width="auto"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <DrawerContent
      :title="panelTitle"
      :closable="true"
      class="w-6xl max-w-[100vw] relative"
    >
      <div class="w-full flex flex-row justify-end items-center">
        <NButton type="primary" @click="state.showAddMemberPanel = true">
          {{ $t("settings.members.grant-access") }}
        </NButton>
      </div>
      <div v-if="binding.type === 'groups'" class="mb-6">
        <div class="text-lg px-1 pb-1 w-full border-b mb-3">
          <GroupNameCell :group="binding.group!" :show-icon="false" />
        </div>
        <div class="border rounded-sm divide-y">
          <div v-for="data in groupMembers" :key="data.user.name" class="p-2">
            <GroupMemberNameCell :user="data.user" :role="data.role" />
          </div>
        </div>
      </div>
      <template v-if="binding.workspaceLevelRoles.size > 0">
        <p class="text-lg px-1 pb-1 w-full border-b mb-3">
          {{ $t("project.members.workspace-level-roles") }}
        </p>
        <div class="flex flex-row items-center flex-wrap gap-2">
          <NTag v-for="role in binding.workspaceLevelRoles" :key="role">
            <template #avatar>
              <NTooltip>
                <template #trigger>
                  <Building2Icon class="w-4 h-auto" />
                </template>
                {{ $t("project.members.workspace-level-roles") }}
              </NTooltip>
            </template>
            {{ displayRoleTitle(role) }}
          </NTag>
        </div>
      </template>
      <template v-if="roleList.length > 0">
        <p
          v-if="binding.workspaceLevelRoles.size > 0"
          class="text-lg px-1 pb-1 w-full border-b mt-4 mb-3"
        >
          {{ $t("project.members.project-level-roles") }}
        </p>
        <div v-for="role in roleList" :key="role.role" class="mb-4">
          <template v-if="role.singleBindingList.length > 0">
            <div
              class="w-full px-2 py-2 flex flex-row justify-start items-center gap-x-1"
            >
              <span class="textlabel">{{ displayRoleTitle(role.role) }}</span>
              <NTooltip
                :disabled="
                  allowRemoveRole(role.role) ||
                  role.role !== PresetRoleType.PROJECT_OWNER
                "
              >
                <template #trigger>
                  <MiniActionButton
                    type="error"
                    :disabled="!allowRemoveRole(role.role)"
                    @click.prevent="handleDeleteRole(role.role)"
                  >
                    <TrashIcon class="w-4 h-4" />
                  </MiniActionButton>
                </template>
                <div>
                  {{ $t("project.members.cannot-remove-last-owner") }}
                </div>
              </NTooltip>
            </div>
            <NDataTable
              size="small"
              :columns="getDataTableColumns(role.role)"
              :data="role.singleBindingList"
              :striped="true"
              :bordered="true"
            />
          </template>
        </div>
      </template>
      <template #footer>
        <div class="w-full flex flex-row justify-between items-center">
          <div>
            <BBButtonConfirm
              :disabled="!allowRevokeMember"
              :type="'DELETE'"
              :confirm-title="$t('settings.members.revoke-access-alert')"
              :ok-text="$t('settings.members.revoke-access')"
              :button-text="$t('settings.members.revoke-access')"
              :require-confirm="true"
              @confirm="$emit('revoke-binding', binding)"
            />
          </div>
          <div class="flex items-center justify-end gap-x-2">
            <NButton @click="$emit('close')">{{ $t("common.cancel") }}</NButton>
            <NButton type="primary" @click="$emit('close')">
              {{ $t("common.ok") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>

  <EditProjectRolePanel
    v-if="editingBinding"
    :project="project"
    :binding="{
      ...editingBinding,
      members: [binding.binding],
    }"
    @close="editingBinding = null"
  />

  <AddProjectMembersPanel
    v-if="state.showAddMemberPanel"
    :project="project"
    :bindings="[
      create(BindingSchema, {
        members: [binding.binding],
      }),
    ]"
    @close="state.showAddMemberPanel = false"
  />
</template>

<script lang="tsx" setup>
import { create } from "@bufbuild/protobuf";
import { computedAsync } from "@vueuse/core";
import { cloneDeep, isEqual } from "lodash-es";
import { Building2Icon, PencilIcon, TrashIcon } from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import { NButton, NDataTable, NTag, NTooltip, useDialog } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBButtonConfirm } from "@/bbkit";
import type { MemberBinding } from "@/components/Member/types";
import GroupMemberNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupMemberNameCell.vue";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import {
  Drawer,
  DrawerContent,
  InstanceV1Name,
  MiniActionButton,
} from "@/components/v2";
import {
  extractGroupEmail,
  extractUserId,
  pushNotification,
  useDatabaseV1Store,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
  useUserStore,
} from "@/store";
import {
  type DatabaseResource,
  PRESET_ROLES,
  PresetRoleType,
  unknownUser,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import { BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  checkRoleContainsAnyPermission,
  displayRoleTitle,
  hasProjectPermissionV2,
  memberMapToRolesInProjectIAM,
} from "@/utils";
import { buildConditionExpr, convertFromExpr } from "@/utils/issue/cel";
import AddProjectMembersPanel from "../AddProjectMember/AddProjectMembersPanel.vue";
import EditProjectRolePanel from "./EditProjectRolePanel.vue";
import RoleDescription from "./RoleDescription.vue";
import RoleExpiredTip from "./RoleExpiredTip.vue";

interface SingleBinding {
  databaseResource?: DatabaseResource;
  expiration?: Date;
  description: string;
  rawBinding: Binding;
}

interface LocalState {
  showAddMemberPanel: boolean;
}

const props = defineProps<{
  project: Project;
  binding: MemberBinding;
}>();

defineEmits<{
  (event: "close"): void;
  (event: "revoke-binding", binding: MemberBinding): void;
}>();

const { t } = useI18n();
const dialog = useDialog();
const userStore = useUserStore();
const databaseStore = useDatabaseV1Store();
const projectIamPolicyStore = useProjectIamPolicyStore();
const projectResourceName = computed(() => props.project.name);
const { policy: iamPolicy } = useProjectIamPolicy(projectResourceName);
const state = reactive<LocalState>({
  showAddMemberPanel: false,
});
const roleList = ref<
  {
    role: string;
    singleBindingList: SingleBinding[];
  }[]
>([]);
const editingBinding = ref<Binding | null>(null);

const panelTitle = computed(() => {
  let email = props.binding.binding;
  if (props.binding.type === "users") {
    email = extractUserId(email);
  } else {
    email = extractGroupEmail(email);
  }
  return t("project.members.edit", {
    member: `${props.binding.title} (${email})`,
  });
});

const allowRevokeMember = computed(() => {
  if (props.binding.projectRoleBindings.length === 0) {
    return false;
  }

  return hasProjectPermissionV2(props.project, "bb.projects.setIamPolicy");
});

const isRoleShouldShowDatabaseRelatedColumns = (role: string) => {
  return (
    role !== PresetRoleType.PROJECT_OWNER &&
    checkRoleContainsAnyPermission(role, "bb.sql.select")
  );
};

const getDataTableColumns = (
  role: string
): DataTableColumn<SingleBinding>[] => {
  const columns: DataTableColumn<SingleBinding>[] = [
    {
      title: t("project.members.condition-name"),
      key: "conditionName",
      render: (singleBinding) => {
        const conditionTitle = singleBinding.rawBinding.condition?.title;
        const roleTitle = displayRoleTitle(singleBinding.rawBinding.role);
        return conditionTitle || roleTitle;
      },
    },
  ];

  if (isRoleShouldShowDatabaseRelatedColumns(role)) {
    columns.push(
      {
        title: t("common.database"),
        key: "database",
        render: (singleBinding) => {
          const databaseName = extractDatabaseName(
            singleBinding.databaseResource
          );
          if (singleBinding.databaseResource) {
            const database = extractDatabase(singleBinding.databaseResource);
            return (
              <div class="flex items-center gap-x-1">
                <InstanceV1Name
                  instance={database.instanceResource}
                  link={false}
                />
                <span>/</span>
                <span>{databaseName}</span>
              </div>
            );
          }
          return databaseName;
        },
      },
      {
        title: t("common.schema"),
        key: "schema",
        render: (singleBinding) =>
          extractSchemaName(singleBinding.databaseResource),
      },
      {
        title: t("common.table"),
        key: "table",
        render: (singleBinding) =>
          extractTableName(singleBinding.databaseResource),
      }
    );
  }

  columns.push(
    {
      title: t("common.expiration"),
      key: "expiration",
      render: (singleBinding) => {
        const content = extractExpiration(singleBinding.expiration);
        if (checkRoleExpired(singleBinding)) {
          return <RoleExpiredTip content={content} />;
        }
        return content;
      },
    },
    {
      title: t("common.description"),
      key: "description",
      render: (singleBinding) => (
        <RoleDescription description={singleBinding.description || ""} />
      ),
    },
    {
      title: "",
      key: "operations",
      width: 32,
      render: (singleBinding) => (
        <div class="flex justify-end pr-2 gap-x-1">
          <MiniActionButton
            onClick={() => {
              editingBinding.value = create(BindingSchema, {
                role: role,
                members: [props.binding.binding],
                condition: singleBinding.rawBinding.condition,
                parsedExpr: singleBinding.rawBinding.parsedExpr,
              });
            }}
          >
            <PencilIcon class="w-4 h-4" />
          </MiniActionButton>
          {(roleList.value.find((r) => r.role === role)?.singleBindingList
            ?.length ?? 0) > 1 && (
            <NTooltip
              disabled={allowDeleteCondition(singleBinding)}
              v-slots={{
                trigger: () => (
                  <MiniActionButton
                    type="error"
                    disabled={!allowDeleteCondition(singleBinding)}
                    onClick={() => {
                      const item = roleList.value.find((r) => r.role === role);
                      if (item) {
                        const index =
                          item.singleBindingList.indexOf(singleBinding);
                        if (index >= 0) {
                          handleDeleteCondition(item, index);
                        }
                      }
                    }}
                  >
                    <TrashIcon class="w-4 h-4" />
                  </MiniActionButton>
                ),
                default: () => t("project.members.cannot-remove-last-owner"),
              }}
            />
          )}
        </div>
      ),
    }
  );

  return columns;
};

// To prevent user accidentally removing roles and lock the project permanently, we take following measures:
// 1. Disallow removing the last OWNER.
// 2. Allow workspace roles who can manage project. This helps when the project OWNER is no longer available.
const allowRemoveRole = (role: string) => {
  if (props.project.state === State.DELETED) {
    return false;
  }
  if (props.binding.type === "groups") {
    return true;
  }

  if (role === PresetRoleType.PROJECT_OWNER) {
    const memberMap = memberMapToRolesInProjectIAM(iamPolicy.value, role);
    // If there is only one owner, disallow removing.
    if (memberMap.size <= 1) {
      return false;
    }
  }

  return true;
};

const handleDeleteRole = (role: string) => {
  const title = t("project.members.revoke-role-from-user", {
    role: displayRoleTitle(role),
    user: props.binding.title,
  });
  dialog.create({
    type: "error",
    title: title,
    content: t("common.cannot-undo-this-action"),
    positiveText: t("common.revoke"),
    negativeText: t("common.cancel"),
    onPositiveClick: async () => {
      const policy = cloneDeep(iamPolicy.value);
      for (const binding of policy.bindings) {
        if (binding.role !== role) {
          continue;
        }
        binding.members = binding.members.filter((member) => {
          return member !== props.binding.binding;
        });
      }
      await projectIamPolicyStore.updateProjectIamPolicy(
        projectResourceName.value,
        policy
      );
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.deleted"),
      });
    },
  });
};

const allowDeleteCondition = (singleBinding: SingleBinding) => {
  if (singleBinding.rawBinding.role === PresetRoleType.PROJECT_OWNER) {
    return allowRemoveRole(PresetRoleType.PROJECT_OWNER);
  }
  return true;
};

const handleDeleteCondition = async (
  item: {
    role: string;
    singleBindingList: SingleBinding[];
  },
  index: number
) => {
  const singleBinding = item.singleBindingList[index];
  const conditionName =
    singleBinding.rawBinding.condition?.title ||
    displayRoleTitle(singleBinding.rawBinding.role);

  const title = t("project.members.revoke-role-from-user", {
    role: conditionName,
    user: props.binding.title,
  });

  dialog.create({
    title: title,
    type: "error",
    content: t("common.cannot-undo-this-action"),
    positiveText: t("common.revoke"),
    negativeText: t("common.cancel"),
    onPositiveClick: async () => {
      const policy = cloneDeep(iamPolicy.value);
      const rawBinding = policy.bindings.find((binding) =>
        isEqual(binding, singleBinding.rawBinding)
      );
      if (!rawBinding) {
        return;
      }
      // Simply remove the member from original binding.
      rawBinding.members = rawBinding.members.filter((member) => {
        return member !== props.binding.binding;
      });

      // Build new bindings with the remaining conditions
      const bindingList = item.singleBindingList.filter((b, i) => {
        if (i === index) {
          return false;
        }
        return isEqual(b.rawBinding, singleBinding.rawBinding);
      });
      if (bindingList.length > 0) {
        const databaseResources = bindingList
          .filter((b) => b.databaseResource)
          .map((b) => b.databaseResource) as DatabaseResource[];

        policy.bindings.push(
          create(BindingSchema, {
            role: item.role,
            members: [props.binding.binding],
            condition: buildConditionExpr({
              role: item.role,
              description: bindingList[0].description,
              expirationTimestampInMS: bindingList[0].expiration?.getTime(),
              databaseResources: databaseResources,
            }),
          })
        );
      }

      await projectIamPolicyStore.updateProjectIamPolicy(
        projectResourceName.value,
        policy
      );
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.deleted"),
      });
    },
  });
};

const extractDatabase = (databaseResource: DatabaseResource) => {
  const database = databaseStore.getDatabaseByName(
    databaseResource.databaseFullName
  );
  return database;
};

const extractDatabaseName = (databaseResource?: DatabaseResource) => {
  if (!databaseResource) {
    return "*";
  }
  const database = extractDatabase(databaseResource);
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
    return t("project.members.never-expires");
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
  () => props.binding,
  async () => {
    const tempRoleList: {
      role: string;
      singleBindingList: SingleBinding[];
    }[] = [];
    for (const rawBinding of props.binding.projectRoleBindings) {
      const singleBindingList = [];
      const singleBinding: SingleBinding = {
        description: rawBinding.condition?.description || "",
        rawBinding: rawBinding,
      };

      if (rawBinding.parsedExpr) {
        const conditionExpr = convertFromExpr(rawBinding.parsedExpr);
        if (conditionExpr.expiredTime) {
          singleBinding.expiration = new Date(conditionExpr.expiredTime);
        }
        if (
          Array.isArray(conditionExpr.databaseResources) &&
          conditionExpr.databaseResources.length > 0
        ) {
          await databaseStore.batchGetOrFetchDatabases(
            conditionExpr.databaseResources.map(
              (resource) => resource.databaseFullName
            )
          );
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

    // Sort by role type.
    tempRoleList.sort((a, b) => {
      if (!PRESET_ROLES.includes(a.role)) return -1;
      if (!PRESET_ROLES.includes(b.role)) return 1;
      return PRESET_ROLES.indexOf(a.role) - PRESET_ROLES.indexOf(b.role);
    });

    roleList.value = tempRoleList;
  },
  { immediate: true, deep: true }
);

const groupMembers = computedAsync(async () => {
  if (props.binding.type !== "groups") {
    return [];
  }

  // Fetch user data for group members
  const members = props.binding.group?.members ?? [];
  await userStore.batchGetOrFetchUsers(members.map((m) => m.member));

  const resp = [];
  for (const member of members) {
    const user =
      userStore.getUserByIdentifier(member.member) ??
      unknownUser(member.member);
    resp.push({
      user,
      role: member.role,
    });
  }
  return resp;
}, []);
</script>
