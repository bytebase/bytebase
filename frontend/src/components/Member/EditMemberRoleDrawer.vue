<template>
  <Drawer @close="$emit('close')">
    <DrawerContent
      class="w-[40rem] max-w-[100vw]"
      :closable="true"
      :title="$t('settings.members.grant-access')"
    >
      <template #default>
        <div class="space-y-4">
          <MembersBindingSelect
            v-if="isCreating"
            v-model:value="state.memberList"
            :required="true"
            :include-all-users="false"
            :include-service-account="true"
          />
          <div v-else class="w-full space-y-2">
            <div class="flex items-center gap-x-1">
              {{ $t("common.email") }}
              <span class="text-red-600">*</span>
            </div>
            <EmailInput :readonly="true" :value="email" />
          </div>

          <div class="w-full space-y-2">
            <div class="flex items-center gap-x-1">
              {{ $t("settings.members.assign-roles") }}
              <span class="text-red-600">*</span>
            </div>
            <NSelect
              v-model:value="state.roles"
              multiple
              :options="availableRoleOptions"
              :placeholder="$t('role.select-roles')"
            />
          </div>
        </div>
      </template>

      <template #footer>
        <div class="w-full flex justify-between items-center">
          <div>
            <NPopconfirm v-if="!isCreating" @positive-click="handleRevoke">
              <template #trigger>
                <NButton quaternary size="small" @click.stop>
                  <template #icon>
                    <ArchiveIcon class="w-4 h-auto" />
                  </template>
                  <template #default>
                    {{ $t("settings.members.action.deactivate") }}
                  </template>
                </NButton>
              </template>

              <template #default>
                <div>
                  {{ $t("settings.members.action.deactivate-confirm-title") }}
                </div>
              </template>
            </NPopconfirm>
          </div>

          <div class="flex flex-row items-center justify-end gap-x-3">
            <NButton @click="$emit('close')">
              {{ $t("common.cancel") }}
            </NButton>
            <NButton
              type="primary"
              :disabled="!allowConfirm"
              :loading="state.isRequesting"
              @click="updateRoleBinding"
            >
              {{ $t("common.confirm") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { ArchiveIcon } from "lucide-vue-next";
import type { SelectGroupOption, SelectOption } from "naive-ui";
import { NPopconfirm, NButton, NSelect } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import EmailInput from "@/components/EmailInput.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  extractGroupEmail,
  extractUserEmail,
  pushNotification,
  useAppFeature,
  useRoleStore,
  useWorkspaceV1Store,
} from "@/store";
import {
  PRESET_PROJECT_ROLES,
  PRESET_ROLES,
  PRESET_WORKSPACE_ROLES,
  PresetRoleType,
} from "@/types";
import { displayRoleTitle } from "@/utils";
import MembersBindingSelect from "./MembersBindingSelect.vue";
import { type MemberBinding } from "./types";

interface LocalState {
  isRequesting: boolean;
  memberList: string[];
  roles: string[];
}

const props = defineProps<{
  member?: MemberBinding;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const initMemberList = () => {
  if (!props.member) {
    return [];
  }
  return [props.member.binding];
};

const state = reactive<LocalState>({
  isRequesting: false,
  memberList: initMemberList(),
  roles: props.member?.workspaceLevelRoles ?? [],
});

const { t } = useI18n();
const hideProjectRoles = useAppFeature("bb.feature.members.hide-project-roles");
const workspaceStore = useWorkspaceV1Store();

const isCreating = computed(() => !props.member);

const email = computed(() => {
  if (!props.member) {
    return "";
  }
  if (props.member.type === "users") {
    return extractUserEmail(props.member.binding);
  }
  return extractGroupEmail(props.member.binding);
});

const availableRoleOptions = computed(
  (): (SelectOption | SelectGroupOption)[] => {
    const roleGroups = [
      {
        type: "group",
        key: "workspace-roles",
        label: t("role.workspace-roles"),
        children: PRESET_WORKSPACE_ROLES.filter(
          (role) => role !== PresetRoleType.WORKSPACE_MEMBER
        ).map((role) => ({
          label: displayRoleTitle(role),
          value: role,
        })),
      },
      {
        type: "group",
        key: "project-roles",
        label: `${t("role.project-roles.self")} (${t("common.optional")}, ${t(
          "role.project-roles.apply-to-all-projects"
        ).toLocaleLowerCase()})`,
        children: PRESET_PROJECT_ROLES.map((role) => ({
          label: displayRoleTitle(role),
          value: role,
        })),
      },
    ];
    if (hideProjectRoles.value) {
      return roleGroups[0].children;
    }
    const customRoles = useRoleStore()
      .roleList.map((role) => role.name)
      .filter((role) => !PRESET_ROLES.includes(role));
    if (customRoles.length > 0) {
      roleGroups.push({
        type: "group",
        key: "custom-roles",
        label: `${t("role.custom-roles")} (${t("common.optional")}, ${t(
          "role.project-roles.apply-to-all-projects"
        ).toLocaleLowerCase()})`,
        children: customRoles.map((role) => ({
          label: displayRoleTitle(role),
          value: role,
        })),
      });
    }
    return roleGroups;
  }
);

const allowConfirm = computed(() => {
  if (state.memberList.length === 0 || state.roles.length === 0) {
    return false;
  }

  return true;
});

const memberListInBinding = computed(() => {
  if (props.member) {
    return [props.member.binding];
  }
  return state.memberList;
});

const updateRoleBinding = async () => {
  const batchPatch = [];
  for (const member of memberListInBinding.value) {
    const existedRoles = workspaceStore.findRolesByMember({
      member,
      ignoreGroup: true,
    });
    batchPatch.push({
      member,
      roles: [...new Set([...state.roles, ...existedRoles])],
    });
  }
  await workspaceStore.patchIamPolicy(batchPatch);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
  emit("close");
};

const handleRevoke = async () => {
  if (!props.member || memberListInBinding.value.length !== 1) {
    return;
  }
  await workspaceStore.patchIamPolicy([
    {
      member: memberListInBinding.value[0],
      roles: [],
    },
  ]);
  pushNotification({
    module: "bytebase",
    style: "INFO",
    title: t("settings.members.revoked"),
  });
  emit("close");
};
</script>
