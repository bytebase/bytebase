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
      <div v-if="binding.workspaceLevelRoles.size > 0" class="mb-6">
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
      </div>
      <div v-if="binding.projectRoleBindings.length > 0">
        <p
          v-if="binding.workspaceLevelRoles.size > 0"
          class="text-lg px-1 pb-1 w-full border-b mt-4 mb-3"
        >
          {{ $t("project.members.project-level-roles") }}
        </p>
        <div v-for="(b, index) in binding.projectRoleBindings" :key="index" class="mb-4 flex flex-col gap-y-2">
          <div>
            <div
              class="w-full flex flex-row justify-start items-center gap-x-1"
            >
              <span class="textlabel">{{ displayRoleTitle(b.role) }}</span>
              <MiniActionButton
                @click="() => {
                  editingBinding = cloneDeep(b)
                }"
              >
                <PencilIcon class="w-4 h-4" />
              </MiniActionButton>
              <MiniActionButton
                type="error"
                :disabled="!allowRemoveRole"
                @click.prevent="handleDeleteRole(b)"
              >
                <TrashIcon class="w-4 h-4" />
              </MiniActionButton>
            </div>
            <span v-if="b.condition?.description" class="textinfolabel">
              {{ b.condition?.description }}
            </span>
          </div>
          <BBAttention v-if="roleHasEnvironmentLimitation(b.role)" :type="'info'">
            <div v-if="getEnvironmentLimitation(b).length > 0">
              <p>
                {{ $t("project.members.allow-ddl") }}
              </p>
              <ul class="list-disc pl-4">
                <li v-for="env in getEnvironmentLimitation(b)" :key="env">
                  {{ env }}
                </li>
              </ul>
            </div>
            <div v-else>
              {{ $t("project.members.disallow-ddl") }}
            </div>
          </BBAttention>
          <NDataTable
            size="small"
            :columns="getDataTableColumns(b.role)"
            :data="getSingleBindingList(b)"
            :striped="true"
            :bordered="true"
          />
          <NDivider />
        </div>
      </div>
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
import {
  NButton,
  NDataTable,
  NDivider,
  NTag,
  NTooltip,
  useDialog,
} from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBAttention, BBButtonConfirm } from "@/bbkit";
import type { MemberBinding } from "@/components/Member/types";
import GroupMemberNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupMemberNameCell.vue";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import { Drawer, DrawerContent, MiniActionButton } from "@/components/v2";
import {
  extractGroupEmail,
  extractUserEmail,
  pushNotification,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
  useUserStore,
} from "@/store";
import { type DatabaseResource, unknownUser } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import { BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  displayRoleTitle,
  formatAbsoluteDateTime,
  hasProjectPermissionV2,
} from "@/utils";
import { convertFromExpr } from "@/utils/issue/cel";
import AddProjectMembersPanel from "../AddProjectMember/AddProjectMembersPanel.vue";
import {
  roleHasDatabaseLimitation,
  roleHasEnvironmentLimitation,
} from "../utils";
import EditProjectRolePanel from "./EditProjectRolePanel.vue";
import RoleExpiredTip from "./RoleExpiredTip.vue";

interface SingleBinding {
  databaseResource?: DatabaseResource;
  expiration?: Date;
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
const projectIamPolicyStore = useProjectIamPolicyStore();
const projectResourceName = computed(() => props.project.name);
const { policy: iamPolicy } = useProjectIamPolicy(projectResourceName);
const state = reactive<LocalState>({
  showAddMemberPanel: false,
});
const editingBinding = ref<Binding | null>(null);

const panelTitle = computed(() => {
  let email = props.binding.binding;
  if (props.binding.type === "users") {
    email = extractUserEmail(email);
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

const getDataTableColumns = (
  role: string
): DataTableColumn<SingleBinding>[] => {
  const columns: DataTableColumn<SingleBinding>[] = [];

  if (roleHasDatabaseLimitation(role)) {
    columns.push(
      {
        title: t("common.database"),
        key: "database",
        render: (singleBinding) => {
          return singleBinding.databaseResource?.databaseFullName ?? "*";
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

  columns.push({
    title: t("common.expiration"),
    key: "expiration",
    render: (singleBinding) => {
      const content = extractExpiration(singleBinding.expiration);
      if (checkRoleExpired(singleBinding)) {
        return <RoleExpiredTip content={content} />;
      }
      return content;
    },
  });

  return columns;
};

const allowRemoveRole = computed(() => {
  if (props.project.state === State.DELETED) {
    return false;
  }
  if (props.binding.type === "groups") {
    return true;
  }

  return true;
});

const handleDeleteRole = (binding: Binding) => {
  const title = t("project.members.revoke-role-from-user", {
    role: displayRoleTitle(binding.role),
    user: props.binding.binding,
  });
  dialog.create({
    type: "error",
    title: title,
    content: t("common.cannot-undo-this-action"),
    positiveText: t("common.revoke"),
    negativeText: t("common.cancel"),
    onPositiveClick: async () => {
      const policy = cloneDeep(iamPolicy.value);
      const rawBinding = policy.bindings.find((b) => isEqual(binding, b));
      if (!rawBinding) {
        return;
      }
      rawBinding.members = binding.members.filter((member) => {
        return member !== props.binding.binding;
      });
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
  return formatAbsoluteDateTime(expiration.getTime());
};

const checkRoleExpired = (role: SingleBinding) => {
  if (!role.expiration) {
    return false;
  }
  return role.expiration < new Date();
};

const getEnvironmentLimitation = (rawBinding: Binding): string[] => {
  if (!rawBinding.parsedExpr) {
    return [];
  }
  const conditionExpr = convertFromExpr(rawBinding.parsedExpr);
  return conditionExpr.environments ?? [];
};

const getSingleBindingList = (rawBinding: Binding): SingleBinding[] => {
  const singleBindingList = [];
  const singleBinding: SingleBinding = {};

  if (rawBinding.parsedExpr) {
    const conditionExpr = convertFromExpr(rawBinding.parsedExpr);
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
  return singleBindingList;
};

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
