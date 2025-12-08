<template>
  <div class="w-full flex flex-col gap-y-2">
    <div
      v-if="allowChangeType"
      class="w-full flex items-center justify-between"
    >
      <NRadioGroup
        :value="memberType"
        class="flex gap-x-2"
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
    <div
      :class="[
        'w-full flex flex-col gap-y-2',
        memberType !== 'USERS' ? 'hidden' : '',
      ]"
    >
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
        :required-permission="requiredPermission"
        :expand-groups="forMaskingExemption"
        :hint="userSelectHint"
        hint-key="masking-exemption-user-select"
        @update:users="onMemberListUpdate"
      />
    </div>
    <div
      :class="[
        'w-full flex flex-col gap-y-2',
        memberType !== 'GROUPS' ? 'hidden' : '',
      ]"
    >
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
        :required-permission="requiredPermission"
        :hint="groupSelectHint"
        hint-key="masking-exemption-group-select"
        @update:groups="onMemberListUpdate"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { uniq } from "lodash-es";
import { NRadio, NRadioGroup } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import RequiredStar from "@/components/RequiredStar.vue";
import { GroupSelect, UserSelect } from "@/components/v2";
import { extractGroupEmail, extractUserId, useGroupStore } from "@/store";
import { groupNamePrefix } from "@/store/modules/v1/common";
import {
  getGroupEmailInBinding,
  getUserEmailInBinding,
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
    // For masking exemption: filter to users with bb.sql.select permission
    forMaskingExemption?: boolean;
  }>(),
  {
    disabled: false,
    projectName: undefined,
    allowChangeType: true,
    includeAllUsers: false,
    includeServiceAccount: false,
    forMaskingExemption: false,
  }
);

const emit = defineEmits<{
  (event: "update:value", memberList: string[]): void;
}>();

const { t } = useI18n();
const memberType = ref<MemberType>("USERS");
const groupStore = useGroupStore();

const requiredPermission = computed(() =>
  props.forMaskingExemption ? "bb.sql.select" : undefined
);

const userSelectHint = computed(() =>
  props.forMaskingExemption
    ? t("settings.sensitive-data.eligible-users-info")
    : undefined
);

const groupSelectHint = computed(() =>
  props.forMaskingExemption
    ? t("settings.sensitive-data.eligible-groups-info")
    : undefined
);

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
      // For users, extract email from binding format "user:{email}"
      const email = extractUserId(binding);
      if (email) {
        list.push(email);
      }
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
      // UserSelect now returns email directly
      return getUserEmailInBinding(member);
    })
    .filter((binding) => binding);

  emit("update:value", memberListInBinding);
};
</script>
