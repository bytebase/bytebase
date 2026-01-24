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
        <NRadio v-if="includeServiceAccount" value="SERVICE_ACCOUNTS">
          {{ $t("settings.members.service-accounts") }}
        </NRadio>
        <NRadio v-if="includeWorkloadIdentity" value="WORKLOAD_IDENTITIES">
          {{ $t("settings.members.workload-identities") }}
        </NRadio>
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
      v-if="memberType === 'USERS'"
      :class="[
        'w-full flex flex-col gap-y-2',
      ]"
    >
      <div class="flex text-main items-center gap-x-1">
        {{ $t("settings.members.select-user", 2 /* multiply*/) }}
        <RequiredStar v-if="required" />
      </div>
      <UserSelect
        key="user-select"
        :value="memberList"
        :multiple="true"
        :disabled="disabled"
        :project-name="projectName"
        :include-all-users="includeAllUsers"
        @update:value="onMemberListUpdate($event as string[])"
      />
    </div>
    <div
      v-else-if="memberType === 'SERVICE_ACCOUNTS'"
      :class="[
        'w-full flex flex-col gap-y-2',
      ]"
    >
      <div class="flex text-main items-center gap-x-1">
        {{ $t("settings.members.select-service-account", 2 /* multiply*/) }}
        <RequiredStar v-if="required" />
      </div>
      <ServiceAccountSelect
        key="service-account-select"
        :value="memberList"
        :multiple="true"
        :disabled="disabled"
        @update:value="onMemberListUpdate($event as string[])"
      />
    </div>
    <div
      v-else-if="memberType === 'WORKLOAD_IDENTITIES'"
      :class="[
        'w-full flex flex-col gap-y-2',
      ]"
    >
      <div class="flex text-main items-center gap-x-1">
        {{ $t("settings.members.select-workload-identity", 2 /* multiply*/) }}
        <RequiredStar v-if="required" />
      </div>
      <WorkloadIdentitySelect
        key="workload-identity-select"
        :value="memberList"
        :multiple="true"
        :disabled="disabled"
        @update:value="onMemberListUpdate($event as string[])"
      />
    </div>
    <div
      v-else-if="memberType === 'GROUPS'"
      :class="[
        'w-full flex flex-col gap-y-2',
      ]"
    >
      <div class="flex text-main items-center gap-x-1">
        {{ $t("settings.members.select-group", 2 /* multiply*/) }}
        <RequiredStar v-if="required" />
      </div>

      <GroupSelect
        key="group-select"
        :value="memberList"
        :disabled="disabled"
        :multiple="true"
        :project-name="projectName"
        @update:value="onMemberListUpdate($event as string[])"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { uniq } from "lodash-es";
import { NRadio, NRadioGroup } from "naive-ui";
import { computed, ref } from "vue";
import RequiredStar from "@/components/RequiredStar.vue";
import {
  GroupSelect,
  ServiceAccountSelect,
  UserSelect,
  WorkloadIdentitySelect,
} from "@/components/v2";
import {
  extractGroupEmail,
  extractServiceAccountId,
  extractUserId,
  extractWorkloadIdentityId,
  groupNamePrefix,
  serviceAccountNamePrefix,
  workloadIdentityNamePrefix,
} from "@/store";
import {
  getGroupEmailInBinding,
  getServiceAccountNameInBinding,
  getUserEmailInBinding,
  getWorkloadIdentityNameInBinding,
  groupBindingPrefix,
  serviceAccountBindingPrefix,
  workloadIdentityBindingPrefix,
} from "@/types";
import { convertMemberToFullname } from "@/utils";

type MemberType =
  | "USERS"
  | "SERVICE_ACCOUNTS"
  | "WORKLOAD_IDENTITIES"
  | "GROUPS";

const props = withDefaults(
  defineProps<{
    // member binding list, could be
    // - user:{email}
    // - group:{email}
    // - serviceAccount:{email}
    // - workloadIdentity:{email}
    // We don't support mix group with other data.
    value: string[];
    required: boolean;
    projectName?: string;
    disabled?: boolean;
    allowChangeType?: boolean;
    includeAllUsers?: boolean;
    includeServiceAccount?: boolean;
    includeWorkloadIdentity?: boolean;
  }>(),
  {
    disabled: false,
    projectName: undefined,
    allowChangeType: true,
    includeAllUsers: false,
    includeServiceAccount: false,
    includeWorkloadIdentity: false,
  }
);

const emit = defineEmits<{
  (event: "update:value", memberList: string[]): void;
}>();

const initMemberType = computed((): MemberType => {
  for (const binding of props.value) {
    if (binding.startsWith(groupBindingPrefix)) {
      return "GROUPS";
    }
    if (binding.startsWith(serviceAccountBindingPrefix)) {
      return "SERVICE_ACCOUNTS";
    }
    if (binding.startsWith(workloadIdentityBindingPrefix)) {
      return "WORKLOAD_IDENTITIES";
    }
    return "USERS";
  }
  return "USERS";
});

const memberType = ref<MemberType>(initMemberType.value);

const onTypeChange = (type: MemberType) => {
  emit("update:value", []);
  memberType.value = type;
};

const memberList = computed(() => {
  return props.value.map(convertMemberToFullname);
});

const convertFullnameToMember = (fullname: string) => {
  if (fullname.startsWith(groupNamePrefix)) {
    const email = extractGroupEmail(fullname);
    return getGroupEmailInBinding(email);
  } else if (fullname.startsWith(serviceAccountNamePrefix)) {
    const email = extractServiceAccountId(fullname);
    return getServiceAccountNameInBinding(email);
  } else if (fullname.startsWith(workloadIdentityNamePrefix)) {
    const email = extractWorkloadIdentityId(fullname);
    return getWorkloadIdentityNameInBinding(email);
  } else {
    const email = extractUserId(fullname);
    return getUserEmailInBinding(email);
  }
};

const onMemberListUpdate = (fullnameList: string[]) => {
  const memberListInBinding = uniq(fullnameList)
    .map(convertFullnameToMember)
    .filter((member) => !!member);

  emit("update:value", memberListInBinding);
};
</script>
