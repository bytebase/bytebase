<template>
  <div class="w-full space-y-2">
    <div
      v-if="allowChangeType"
      class="w-full flex items-center justify-between"
    >
      <NRadioGroup
        :value="memberType"
        class="space-x-2"
        :disabled="disabled"
        @update:value="onTypeChange"
      >
        <NRadio value="USERS">{{ $t("common.users") }}</NRadio>
        <NRadio value="GROUPS">
          {{ $t("common.groups") }}
        </NRadio>
      </NRadioGroup>
      <slot name="suffix" />
    </div>
    <i18n-t
      v-if="projectName"
      keypath="settings.members.select-in-project"
      tag="span"
      class="textinfolabel"
    >
      <template #project>
        <span class="font-semibold">{{ projectName }}</span>
      </template>
    </i18n-t>
    <div :class="['w-full space-y-2', memberType !== 'USERS' ? 'hidden' : '']">
      <div class="flex text-main items-center gap-x-1">
        {{ $t("settings.members.select-user", 2 /* multiply*/) }}
        <RequiredStar v-if="required" />
      </div>
      <UserSelect
        key="user-select"
        :users="memberList"
        :multiple="true"
        :disabled="disabled"
        :project-name="projectName"
        :include-all-users="includeAllUsers"
        :include-service-account="includeServiceAccount"
        @update:users="onMemberListUpdate"
      />
    </div>
    <div :class="['w-full space-y-2', memberType !== 'GROUPS' ? 'hidden' : '']">
      <div class="flex font-medium text-main items-center gap-x-1">
        {{ $t("settings.members.select-group", 2 /* multiply*/) }}
        <RequiredStar v-if="required" />
      </div>

      <GroupSelect
        key="group-select"
        :groups="memberList"
        :disabled="disabled"
        :multiple="true"
        :project-name="projectName"
        @update:groups="onMemberListUpdate"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { uniq } from "lodash-es";
import { NRadio, NRadioGroup } from "naive-ui";
import { computed, ref, onMounted, watchEffect } from "vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { GroupSelect, UserSelect } from "@/components/v2";
import {
  extractGroupEmail,
  useUserStore,
  useGroupStore,
  extractUserId,
} from "@/store";
import { groupNamePrefix } from "@/store/modules/v1/common";
import {
  getUserEmailInBinding,
  getGroupEmailInBinding,
  groupBindingPrefix,
} from "@/types";

type MemberType = "USERS" | "GROUPS";

const props = withDefaults(
  defineProps<{
    // member binding list, for users, should be user:{email}, for groups, shoud be groups:{email}
    // We don't support mixed data.
    value: string[];
    required: boolean;
    projectName?: string;
    disabled?: boolean;
    allowChangeType?: boolean;
    includeAllUsers?: boolean;
    includeServiceAccount?: boolean;
  }>(),
  {
    disabled: false,
    projectName: undefined,
    allowChangeType: true,
    includeAllUsers: false,
    includeServiceAccount: false,
  }
);

const emit = defineEmits<{
  (event: "update:value", memberList: string[]): void;
}>();

const memberType = ref<MemberType>("USERS");
const userStore = useUserStore();
const groupStore = useGroupStore();

onMounted(async () => {
  const isGroupType =
    props.value.length > 0 &&
    props.value.every((member) => member.startsWith(groupBindingPrefix));
  memberType.value = isGroupType ? "GROUPS" : "USERS";

  // This component needs ALL groups for selection when user switches to group mode
  // AuthContext only loads groups that have IAM policy bindings via batchFetchGroups
  await groupStore.fetchGroupList();
});

watchEffect(async () => {
  await userStore.batchGetUsers(
    props.value.map((binding) =>
      binding.startsWith(groupBindingPrefix) ? "" : binding
    )
  );
});

const onTypeChange = (type: MemberType) => {
  emit("update:value", []);
  memberType.value = type;
};

const memberList = computed(() => {
  const list = [];

  for (const binding of props.value) {
    if (binding.startsWith(groupBindingPrefix)) {
      const group = groupStore.getGroupByIdentifier(binding);
      if (!group) {
        continue;
      }
      list.push(group.name);
    } else {
      const user = userStore.getUserByIdentifier(binding);
      if (!user) {
        continue;
      }
      list.push(extractUserId(user.name));
    }
  }

  return list;
});

const onMemberListUpdate = (memberList: string[]) => {
  const memberListInBinding = uniq(memberList)
    .map((member) => {
      if (member.startsWith(groupNamePrefix)) {
        const email = extractGroupEmail(member);
        return getGroupEmailInBinding(email);
      }
      const user = userStore.getUserByIdentifier(member);
      if (!user) {
        return "";
      }
      return getUserEmailInBinding(user.email);
    })
    .filter((binding) => binding);

  emit("update:value", memberListInBinding);
};
</script>
