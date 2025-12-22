<template>
  <Drawer @close="$emit('close')">
    <DrawerContent
      class="w-200 max-w-[90vw] relative"
      :title="
        isCreating
          ? $t('settings.members.groups.add-group')
          : $t('settings.members.groups.update-group')
      "
    >
      <template #default>
        <div class="flex flex-col gap-y-4">
          <div class="flex flex-col gap-y-2">
            <div>
              <label class="textlabel block">
                {{ $t("settings.members.groups.form.email") }}
                <RequiredStar />
              </label>
              <span class="textinfolabel">
                {{ $t("settings.members.groups.form.email-tips") }}
              </span>
            </div>
            <EmailInput
              v-model:value="state.group.email"
              :readonly="!isCreating"
              :show-domain="true"
            />
          </div>
          <div class="flex flex-col gap-y-2">
            <label class="textlabel block">
              {{ $t("settings.members.groups.form.title") }}
              <RequiredStar />
            </label>
            <NInput
              v-model:value="state.group.title"
              :disabled="!allowEdit"
              :maxlength="200"
            />
          </div>
          <div class="flex flex-col gap-y-2">
            <label class="textlabel block">
              {{ $t("settings.members.groups.form.description") }}
            </label>
            <NInput
              v-model:value="state.group.description"
              :disabled="!allowEdit"
              :maxlength="1000"
            />
          </div>
          <div class="flex flex-col gap-y-2">
            <label class="textlabel block">
              {{ $t("settings.members.groups.form.members") }}
            </label>
            <div class="flex flex-col gap-y-2">
              <div
                v-for="(member, i) in state.group.members"
                :key="i"
                class="w-full flex items-center gap-x-3"
              >
                <UserSelect
                  :value="extractUserId(member.member)"
                  :multiple="false"
                  :size="'medium'"
                  :include-all="false"
                  :disabled="!allowEdit"
                  :filter="(user) => userFilter(user, member.member)"
                  @update:value="($event) => updateMemberEmail(i, $event as (string | undefined))"
                />
                <GroupMemberRoleSelect
                  :value="member.role"
                  :size="'medium'"
                  :disabled="!allowEdit"
                  @update:value="(role) => updateMemberRole(i, role)"
                />
                <div class="flex justify-end">
                  <NButton
                    quaternary
                    size="tiny"
                    type="error"
                    :disabled="!allowEdit"
                    @click="deleteMember(i)"
                  >
                    <template #icon>
                      <Trash2Icon class="w-4 h-auto" />
                    </template>
                  </NButton>
                </div>
              </div>
            </div>
            <div>
              <NButton
                :disabled="!allowEdit"
                @click="addMember"
              >
                {{ $t("settings.members.add-member") }}
              </NButton>
            </div>
          </div>
        </div>
      </template>
      <template #footer>
        <div class="w-full flex justify-between items-center">
          <RemoveGroupButton
            v-if="!isCreating && allowDelete"
            quaternary
            size="small"
            :group="group!"
            @removed="$emit('removed', group!)"
          >
            <template #icon>
              <Trash2Icon class="w-4 h-auto" />
            </template>
            <template #default>
              {{ $t("common.delete") }}
            </template>
          </RemoveGroupButton>

          <div class="flex-1 flex flex-row items-center justify-end gap-x-2">
            <NButton @click="$emit('close')">
              {{ $t("common.cancel") }}
            </NButton>
            <NTooltip :disabled="!errorMessage">
              <template #trigger>
                <NButton
                  type="primary"
                  :disabled="!allowEdit || !allowConfirm"
                  :loading="state.isRequesting"
                  @click="tryCreateOrUpdateGroup"
                >
                  {{ $t("common.confirm") }}
                </NButton>
              </template>
              <span class="w-56 text-sm">
                {{ errorMessage }}
              </span>
            </NTooltip>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { cloneDeep, isEqual } from "lodash-es";
import { Trash2Icon } from "lucide-vue-next";
import { NButton, NInput, NTooltip } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import EmailInput from "@/components/EmailInput.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { Drawer, DrawerContent, UserSelect } from "@/components/v2";
import { pushNotification, useCurrentUserV1, useGroupStore } from "@/store";
import {
  extractUserId,
  groupNamePrefix,
  userNamePrefix,
} from "@/store/modules/v1/common";
import type { Group, GroupMember } from "@/types/proto-es/v1/group_service_pb";
import {
  GroupMember_Role,
  GroupMemberSchema,
  GroupSchema,
} from "@/types/proto-es/v1/group_service_pb";
import { type User } from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2, isValidEmail } from "@/utils";
import RemoveGroupButton from "./RemoveGroupButton.vue";
import GroupMemberRoleSelect from "./UserDataTableByGroup/cells/GroupMemberRoleSelect.vue";

interface LocalState {
  isRequesting: boolean;
  group: {
    email: string;
    title: string;
    description: string;
    members: GroupMember[];
  };
}

const props = defineProps<{
  group?: Group;
}>();

const emit = defineEmits<{
  (event: "close"): void;
  (event: "removed", group: Group): void;
  (event: "updated", group: Group): void;
}>();

const { t } = useI18n();
const groupStore = useGroupStore();
const currentUserV1 = useCurrentUserV1();

const state = reactive<LocalState>({
  isRequesting: false,
  group: {
    email: props.group?.email ?? "",
    title: props.group?.title ?? "",
    description: props.group?.description ?? "",
    members: cloneDeep(
      props.group?.members ?? [
        create(GroupMemberSchema, {
          role: GroupMember_Role.OWNER,
          member: `${userNamePrefix}${currentUserV1.value.email}`,
        }),
      ]
    ),
  },
});

const userFilter = (user: User, member: string) => {
  if (extractUserId(member) === user.email) {
    return true;
  }
  return !state.group.members.find(
    (member) => member.member === `${userNamePrefix}${user.email}`
  );
};

const isCreating = computed(() => !props.group);

const isGroupOwner = computed(() => {
  return (
    props.group?.members.find(
      (member) => extractUserId(member.member) === currentUserV1.value.email
    )?.role === GroupMember_Role.OWNER
  );
});

const allowDelete = computed(() => {
  return isGroupOwner.value || hasWorkspacePermissionV2("bb.groups.delete");
});

const allowEdit = computed(() => {
  if (!!props.group?.source) {
    // do not support edit external group manually.
    return false;
  }
  if (isGroupOwner.value) {
    return true;
  }
  return hasWorkspacePermissionV2(
    isCreating.value ? "bb.groups.create" : "bb.groups.update"
  );
});

const validGroup = computed(() => {
  const memberMap = new Map<string, GroupMember>();
  for (const member of state.group.members) {
    if (!member.member) {
      continue;
    }
    if (
      !memberMap.has(member.member) ||
      member.role === GroupMember_Role.OWNER
    ) {
      memberMap.set(member.member, member);
    }
  }
  return create(GroupSchema, {
    name: props.group
      ? props.group.name
      : `${groupNamePrefix}${state.group.email}`,
    title: state.group.title,
    description: state.group.description,
    members: [...memberMap.values()],
  });
});

const errorMessage = computed(() => {
  if (!isValidEmail(state.group.email)) {
    return "Invalid group email";
  }
  if (!state.group.title) {
    return "Title is required";
  }
  return "";
});

const allowConfirm = computed(() => {
  if (errorMessage.value) {
    return false;
  }

  return !isEqual(props.group, validGroup.value);
});

const addMember = () => {
  const member = create(GroupMemberSchema, {
    role:
      state.group.members.length === 0
        ? GroupMember_Role.OWNER
        : GroupMember_Role.MEMBER,
  });
  state.group.members.push(member);
};

const updateMemberEmail = (index: number, email: string | undefined) => {
  if (!email) {
    return;
  }
  state.group.members[index] = {
    ...state.group.members[index],
    member: `${userNamePrefix}${email}`,
  };
};

const updateMemberRole = (index: number, role: GroupMember_Role) => {
  state.group.members[index] = {
    ...state.group.members[index],
    role,
  };
};

const deleteMember = (index: number) => {
  state.group.members.splice(index, 1);
};

const tryCreateOrUpdateGroup = async () => {
  state.isRequesting = true;

  try {
    let group;
    if (isCreating.value) {
      group = await groupStore.createGroup(validGroup.value);
    } else {
      group = await groupStore.updateGroup(validGroup.value);
    }
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: isCreating.value ? t("common.created") : t("common.updated"),
    });
    emit("close");
    if (group) {
      emit("updated", group);
    }
  } finally {
    state.isRequesting = false;
  }
};
</script>
